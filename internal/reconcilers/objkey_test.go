package reconcile

import "sigs.k8s.io/controller-runtime/pkg/client"

func client_objKey(ns, name string) client.ObjectKey {
	return client.ObjectKey{Namespace: ns, Name: name}
}
