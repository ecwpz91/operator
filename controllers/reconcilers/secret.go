package reconcilers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/bcmendoza/gm-operator/api/v1"
	"github.com/bcmendoza/gm-operator/controllers/gmcore"
)

type Secret struct {
	ObjectKey     types.NamespacedName
	ObjectLiteral *corev1.Secret
}

func (s Secret) Kind() string {
	return "Secret"
}

func (s Secret) Key() types.NamespacedName {
	return s.ObjectKey
}

func (s Secret) Object() client.Object {
	return &corev1.Secret{}
}

func (s Secret) Build(mesh *v1.Mesh, _ gmcore.Configs) client.Object {
	return s.ObjectLiteral
}

func (s Secret) Reconciled(mesh *v1.Mesh, _ gmcore.Configs, obj client.Object) (bool, error) {
	return true, nil
}

func (s Secret) Mutate(mesh *v1.Mesh, _ gmcore.Configs, obj client.Object) client.Object {
	return obj
}
