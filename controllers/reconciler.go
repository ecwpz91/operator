package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/bcmendoza/gm-operator/api/v1"
	"github.com/bcmendoza/gm-operator/controllers/gmcore"
)

type reconciler interface {
	// Returns a string describing the kind of the object being reconciled
	Kind() string
	// Returns the object key used to retrieve the object from the Kubernetes cluster.
	// If the object is cluster-scoped, the 'Namespace' field is an empty string.
	Key() types.NamespacedName
	// Returns an object that implements the client.Object interface (e.g. *appsv1.Deployment).
	Object() client.Object
	// Generates a client.Object with configuration from a *v1.Mesh and gmcore.Configs.
	// The client.Object parameter allows for modifying mutable values of an existing object.
	Reconcile(*v1.Mesh, gmcore.Configs, client.Object) client.Object
}

func apply(ctx context.Context, controller *MeshController, mesh *v1.Mesh, configs gmcore.Configs, r reconciler) error {
	key := r.Key()

	logger := controller.Log.WithValues("Name", key.Name)
	if key.Namespace != "" {
		logger = logger.WithValues("Namespace", key.Namespace)
	}

	obj := r.Object()
	if err := controller.Get(ctx, key, obj); err != nil {
		if errors.IsNotFound(err) {
			obj = r.Reconcile(mesh, configs, obj)
			ctrl.SetControllerReference(mesh, obj, controller.Scheme)
			if err = controller.Create(ctx, obj); err != nil {
				logger.Error(err, "Create "+r.Kind()+" failed")
				return err
			}
			logger.Info("Created " + r.Kind())
			return nil
		} else {
			logger.Error(err, "Get "+r.Kind()+" failed")
			return err
		}
	}

	obj = r.Reconcile(mesh, configs, obj)
	obj.SetManagedFields(nil)
	patchOpt := &client.PatchOptions{FieldManager: "gm-operator"}
	if err := controller.Patch(ctx, obj, client.Apply, patchOpt); err != nil {
		logger.Error(err, "Patch "+r.Kind()+" failed")
		return err
	}
	logger.Info("Patched " + r.Kind())

	return nil
}
