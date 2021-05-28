package controllers

import (
	"context"
	"fmt"

	installv1 "github.com/bcmendoza/gm-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *MeshReconciler) mkControlAPI(ctx context.Context, mesh *installv1.Mesh) error {

	// Check if the deployment exists; if not, create a new one
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: "control-api", Namespace: mesh.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment = r.mkControlAPIDeployment(mesh)
		r.Log.Info("Creating deployment", "Name", "control-api", "Namespace", mesh.Namespace)
		err = r.Create(ctx, deployment)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Failed to create deployment for %s:control-api", mesh.Namespace))
			return err
		}
	} else if err != nil {
		r.Log.Error(err, fmt.Sprintf("Failed to get deployment for %s:control-api", mesh.Namespace))
		return err
	}

	// Check if the service exists; if not, create a new one
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "control-api", Namespace: mesh.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// TODO: Create service
	} else if err != nil {
		r.Log.Error(err, fmt.Sprintf("failed to get corev1.Service for %s:control-api", mesh.Namespace))
	}

	// TODO: Configure mesh objects (send requests to service)
	// Check if objects exist; if not, create them

	return nil
}

func (r *MeshReconciler) mkControlAPIDeployment(mesh *installv1.Mesh) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{
		"deployment":            "control-api",
		"greymatter":            "fabric",
		"greymatter.io/control": "control-api",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-api",
			Namespace: mesh.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "docker.secret"},
					},
					DNSPolicy:     corev1.DNSClusterFirst,
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:  "control-api",
							Image: "docker.greymatter.io/development/gm-control-api:1.6.0-rc.1",
							Env: []corev1.EnvVar{
								{Name: "GM_CONTROL_API_ADDRESS", Value: "0.0.0.0:5555"},
								{Name: "GM_CONTROL_API_DISABLE_VERSION_CHECK", Value: "false"},
								{Name: "GM_CONTROL_API_LOG_LEVEL", Value: "debug"},
								{Name: "GM_CONTROL_API_PERSISTER_TYPE", Value: "null"},
								{Name: "GM_CONTROL_API_EXPERIMENTS", Value: "true"},
								{Name: "GM_CONTROL_API_BASE_URL", Value: "/services/control-api/latest/v1.0/"},
								{Name: "GM_CONTROL_API_USE_TLS", Value: "false"},
								{Name: "GM_CONTROL_API_ORG_KEY", Value: "deciphernow"},
								{Name: "GM_CONTROL_API_ZONE_KEY", Value: "zone-default-zone"},
								{Name: "GM_CONTROL_API_ZONE_NAME", Value: "zone-default-zone"},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 5555, Name: "http", Protocol: "TCP"},
							},
						},
						{
							Name:  "sidecar",
							Image: "docker.greymatter.io/development/gm-proxy:1.6.0-rc.1",
							Env: []corev1.EnvVar{
								{Name: "ENVOY_ADMIN_LOG_PATH", Value: "/dev/stdout"},
								{Name: "PROXY_DYNAMIC", Value: "true"},
								{Name: "XDS_CLUSTER", Value: "control-api"},
								{Name: "XDS_HOST", Value: fmt.Sprintf("control.%s.svc", mesh.Namespace)},
								{Name: "XDS_PORT", Value: "50000"},
								{Name: "XDS_ZONE", Value: "zone-default-zone"},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 10808, Name: "proxy", Protocol: "TCP"},
								{ContainerPort: 8081, Name: "metrics", Protocol: "TCP"},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(mesh, deployment, r.Scheme)
	return deployment
}

func (r *MeshReconciler) mkControlAPIService(mesh *installv1.Mesh) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{
		"deployment":            "control-api",
		"greymatter":            "fabric",
		"greymatter.io/control": "control-api",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-api",
			Namespace: mesh.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "docker.secret"},
					},
					DNSPolicy:     corev1.DNSClusterFirst,
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:  "control-api",
							Image: "docker.greymatter.io/development/gm-control-api:1.6.0-rc.1",
							Env: []corev1.EnvVar{
								{Name: "GM_CONTROL_API_ADDRESS", Value: "0.0.0.0:5555"},
								{Name: "GM_CONTROL_API_DISABLE_VERSION_CHECK", Value: "false"},
								{Name: "GM_CONTROL_API_LOG_LEVEL", Value: "debug"},
								{Name: "GM_CONTROL_API_PERSISTER_TYPE", Value: "null"},
								{Name: "GM_CONTROL_API_EXPERIMENTS", Value: "true"},
								{Name: "GM_CONTROL_API_BASE_URL", Value: "/services/control-api/latest/v1.0/"},
								{Name: "GM_CONTROL_API_USE_TLS", Value: "false"},
								{Name: "GM_CONTROL_API_ORG_KEY", Value: "deciphernow"},
								{Name: "GM_CONTROL_API_ZONE_KEY", Value: "zone-default-zone"},
								{Name: "GM_CONTROL_API_ZONE_NAME", Value: "zone-default-zone"},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 5555, Name: "http", Protocol: "TCP"},
							},
						},
						{
							Name:  "sidecar",
							Image: "docker.greymatter.io/development/gm-proxy:1.6.0-rc.1",
							Env: []corev1.EnvVar{
								{Name: "ENVOY_ADMIN_LOG_PATH", Value: "/dev/stdout"},
								{Name: "PROXY_DYNAMIC", Value: "true"},
								{Name: "XDS_CLUSTER", Value: "control-api"},
								{Name: "XDS_HOST", Value: fmt.Sprintf("control.%s.svc", mesh.Namespace)},
								{Name: "XDS_PORT", Value: "50000"},
								{Name: "XDS_ZONE", Value: "zone-default-zone"},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 10808, Name: "proxy", Protocol: "TCP"},
								{ContainerPort: 8081, Name: "metrics", Protocol: "TCP"},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(mesh, deployment, r.Scheme)
	return deployment
}