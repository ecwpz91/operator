package reconcilers

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/bcmendoza/gm-operator/api/v1"
	"github.com/bcmendoza/gm-operator/controllers/gmcore"
)

type ClusterRoleBinding struct {
	Name string
}

func (crb ClusterRoleBinding) Kind() string {
	return "ClusterRoleBinding"
}

func (crb ClusterRoleBinding) Key() types.NamespacedName {
	return types.NamespacedName{Name: crb.Name}
}

func (crb ClusterRoleBinding) Object() client.Object {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: crb.Name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     crb.Name,
		},
	}
}

func (crb ClusterRoleBinding) Reconcile(mesh *v1.Mesh, _ gmcore.Configs, obj client.Object) client.Object {
	binding := obj.(*rbacv1.ClusterRoleBinding)

	var hasSubject bool
	for _, subject := range binding.Subjects {
		if subject.Name == crb.Name && subject.Namespace == mesh.Namespace {
			hasSubject = true
		}
	}

	if !hasSubject {
		binding.Subjects = append(binding.Subjects, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      crb.Name,
			Namespace: mesh.Namespace,
		})
	}

	return binding
}
