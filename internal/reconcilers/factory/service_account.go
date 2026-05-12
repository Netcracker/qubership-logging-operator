package build

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewServiceAccount returns a *corev1.ServiceAccount with the standard operator label
// set. Callers can attach ImagePullSecrets via the returned object before passing to
// reconcile.ServiceAccount.
func NewServiceAccount(name, namespace, component string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta:   serviceAccountTypeMeta(),
		ObjectMeta: ObjectMeta(name, namespace, component),
	}
}

func serviceTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{Kind: "Service", APIVersion: "v1"}
}

func serviceAccountTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{Kind: "ServiceAccount", APIVersion: "v1"}
}
