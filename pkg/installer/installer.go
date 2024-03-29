// Package installer exposes functions for applying resources to a Kubernetes cluster.
// Its exposed functions receive a k8sClient for communicating with the cluster.
package installer

import (
	"context"
	configv1 "github.com/openshift/api/config/v1"
	"time"

	"github.com/greymatter-io/operator/api/v1alpha1"
	"github.com/greymatter-io/operator/pkg/cfsslsrv"
	"github.com/greymatter-io/operator/pkg/cli"
	"github.com/greymatter-io/operator/pkg/cuemodule"
	"github.com/greymatter-io/operator/pkg/k8sapi"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = ctrl.Log.WithName("installer")
)

// Installer stores a map of version.Version and a distinct version.Sidecar for each mesh.
type Installer struct {
	*cli.CLI  // Grey Matter CLI
	k8sClient *client.Client

	cfssl *cfsslsrv.CFSSLServer

	// The meshes.greymatter.io CRD, used as an owner when applying cluster-scoped resources.
	// If the operator is uninstalled on a cluster, owned cluster-scoped resources will be cleaned up.
	owner *extv1.CustomResourceDefinition
	// The Docker image pull secret to create in namespaces where core services are installed.
	imagePullSecret *corev1.Secret
	// The name of a configured cluster ingress name for OpenShift environments.
	clusterIngressName string

	// Container for THE mesh (on the way to an experimental 1:1 operator:mesh paradigm)
	// Contains the default after load
	Mesh *v1alpha1.Mesh

	// Container for all K8s and GM CUE cue.Values
	OperatorCUE *cuemodule.OperatorCUE

	CueRoot string

	// The cluster ingress domain
	clusterIngressDomain string
}

// New returns a new *Installer instance for installing Grey Matter components and dependencies.
func New(c *client.Client, operatorCUE *cuemodule.OperatorCUE, initialMesh *v1alpha1.Mesh, cueRoot string, gmcli *cli.CLI, cfssl *cfsslsrv.CFSSLServer, clusterIngressName string) (*Installer, error) {
	return &Installer{
		CLI:                gmcli,
		k8sClient:          c,
		cfssl:              cfssl,
		clusterIngressName: clusterIngressName,
		OperatorCUE:        operatorCUE,
		Mesh:               initialMesh,
		CueRoot:            cueRoot,
	}, nil
}

// Start initializes resources and configurations after controller-manager has launched.
// It implements the controller-runtime Runnable interface.
func (i *Installer) Start(ctx context.Context) error {

	// Retrieve the operator image secret from the apiserver (block until it's retrieved).
	// This secret will be re-created in each install namespace and watch namespaces where core services are pulled.
	i.imagePullSecret = getImagePullSecret(i.k8sClient)

	// Get our Mesh CRD to set as an owner for cluster-scoped resources
	i.owner = &extv1.CustomResourceDefinition{}
	err := (*i.k8sClient).Get(ctx, client.ObjectKey{Name: "meshes.greymatter.io"}, i.owner)
	if err != nil {
		logger.Error(err, "Failed to get CustomResourceDefinition meshes.greymatter.io")
		return err
	}

	logger.Info("Attempting to apply spire server-ca secret")
	spireSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "server-ca",
			Namespace: "spire",
		},
	}
	spireSecret, err = injectPKI(spireSecret, i.cfssl)
	if err != nil {
		logger.Error(err, "Error while attempting to apply spire server-ca secret", "secret object", spireSecret)
		return err
	}
	k8sapi.Apply(i.k8sClient, spireSecret, i.owner, k8sapi.CreateOrUpdate)

	// TODO bring this back when we re-enable spire and mTLS (but pull it into the new_structure CUE, and only apply if yes spire)
	//// Ensure our cluster-scoped RBAC permissions and SPIRE resources are created.
	//applyClusterRBAC(i.k8sClient, i.owner)
	//if i.cfssl != nil {
	//	applySpire(i.k8sClient, i.owner, i.cfssl)
	//}

	// TODO bring this back when you get to re-enabling (testing with) OpenShift
	// Try to get the OpenShift cluster ingress domain if it exists.
	clusterIngressDomain, ok := getOpenshiftClusterIngressDomain(i.k8sClient, i.clusterIngressName)
	if ok {
		// TODO: When not in OpenShift, check for other supported ingress class types such as Nginx or Voyager.
		// If no supported ingress types are found, just assume the user will configure ingress on their own.
		logger.Info("Identified OpenShift cluster domain name", "Domain", clusterIngressDomain)
		i.clusterIngressDomain = clusterIngressDomain
	}

	// DEBUG Immediately apply the default mesh from the CUE
	// TODO do we want to keep this, maybe with a flag in the CUE?
	// Create it
	go func() {
		time.Sleep(100 * time.Second) // DEBUG - does this work if we wait long enough for it to register the webhooks?
		k8sapi.Apply(i.k8sClient, i.Mesh, nil, k8sapi.CreateOrUpdate)
	}()
	//// Look it back up to get its ID (for use as an owner of other resources)
	//meshList := &v1alpha1.MeshList{}
	//if err := i.k8sClient.List(context.TODO(), meshList); err != nil {
	//	return err
	//}
	//if len(meshList.Items) > 0 {
	//	for _, mesh := range meshList.Items {
	//		i.Mesh = &mesh
	//	}
	//	go i.ApplyMesh(nil, i.Mesh)
	//}

	return nil
}

// Retrieves the image pull secret in the gm-operator namespace.
// This retries indefinitely at 30s intervals and will block by design.
func getImagePullSecret(c *client.Client) *corev1.Secret {
	key := client.ObjectKey{Name: "gm-docker-secret", Namespace: "gm-operator"}
	operatorSecret := &corev1.Secret{}
	for operatorSecret.CreationTimestamp.IsZero() {
		if err := (*c).Get(context.TODO(), key, operatorSecret); err != nil {
			logger.Error(err, "No 'gm-docker-secret' image pull secret found in gm-operator namespace. Will retry in 30s.")
			time.Sleep(time.Second * 30)
		}
	}

	// Return new secret with just the dockercfgjson (without additional metadata).
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gm-docker-secret"},
		Type:       operatorSecret.Type,
		Data:       operatorSecret.Data,
	}
}

// TODO bring this back when we re-enable OpenShift support
func getOpenshiftClusterIngressDomain(c *client.Client, ingressName string) (string, bool) {
	clusterIngressList := &configv1.IngressList{}
	if err := (*c).List(context.TODO(), clusterIngressList); err != nil {
		return "", false
	} else {
		for _, i := range clusterIngressList.Items {
			if i.Name == ingressName {
				return i.Spec.Domain, true
			}
		}
	}
	return "", false
}

// Check that a suported ingress controller class exists in a kubernetes cluster.
// This will be expanded later on as we support additional ingress implementations.
//lint:ignore U1000 save for reference
func isSupportedKubernetesIngressClassPresent(c client.Client) bool {
	ingressClassList := &networkingv1.IngressClassList{}
	if err := c.List(context.TODO(), ingressClassList); err != nil {
		return false
	}
	for _, i := range ingressClassList.Items {
		switch i.Spec.Controller {
		case "nginx", "voyager":
			return true
		}
	}
	return false
}
