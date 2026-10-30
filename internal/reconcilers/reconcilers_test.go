package reconcilers

import (
	"context"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const restartedAt = "kubectl.kubernetes.io/restartedAt"

func newClient(t *testing.T, objects ...client.Object) (client.Client, *runtime.Scheme, *loggingService.LoggingService) {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1.AddToScheme, loggingService.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatalf("add to scheme: %v", err)
		}
	}
	cr := &loggingService.LoggingService{ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging", UID: "uid"}}
	objects = append(objects, cr)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(), scheme, cr
}

func daemonSet(labels, podAnnotations map[string]string, image string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "logging", Labels: labels},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": "agent"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"name": "agent"}, Annotations: podAnnotations},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: image}}},
			},
		},
	}
}

func TestDaemonSetCreateSetsOwner(t *testing.T) {
	c, scheme, cr := newClient(t)
	if err := DaemonSet(context.Background(), c, scheme, cr, daemonSet(nil, nil, "agent:1")); err != nil {
		t.Fatalf("create DaemonSet: %v", err)
	}
	created := &appsv1.DaemonSet{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "logging", Name: "agent"}, created); err != nil {
		t.Fatalf("get DaemonSet: %v", err)
	}
	if owner := metav1.GetControllerOf(created); owner == nil || owner.UID != cr.UID {
		t.Errorf("controller owner = %v, want the LoggingService", owner)
	}
}

func TestDaemonSetUpdateKeepsForeignLabelsAndPodAnnotations(t *testing.T) {
	existing := daemonSet(map[string]string{"foreign": "kept", "app": "old"}, map[string]string{restartedAt: "now"}, "agent:1")
	c, scheme, cr := newClient(t, existing)

	desired := daemonSet(map[string]string{"app": "new"}, nil, "agent:2")
	if err := DaemonSet(context.Background(), c, scheme, cr, desired); err != nil {
		t.Fatalf("update DaemonSet: %v", err)
	}

	updated := &appsv1.DaemonSet{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "logging", Name: "agent"}, updated); err != nil {
		t.Fatalf("get DaemonSet: %v", err)
	}
	if updated.Labels["foreign"] != "kept" || updated.Labels["app"] != "new" {
		t.Errorf("labels = %v, want the foreign label kept and app=new", updated.Labels)
	}
	if updated.Spec.Template.Annotations[restartedAt] != "now" {
		t.Errorf("pod template annotations = %v, want %s kept to avoid a rollout", updated.Spec.Template.Annotations, restartedAt)
	}
	if got := updated.Spec.Template.Spec.Containers[0].Image; got != "agent:2" {
		t.Errorf("image = %q, want agent:2", got)
	}
}

func TestServiceUpdateKeepsClusterIP(t *testing.T) {
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "logging", Labels: map[string]string{"foreign": "dropped"}},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.10",
			Ports:     []corev1.ServicePort{{Name: "http", Port: 2020, TargetPort: intstr.FromInt32(2020)}},
		},
	}
	c, scheme, cr := newClient(t, existing)

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "logging", Labels: map[string]string{"app": "agent"}},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Name: "metrics", Port: 2021, TargetPort: intstr.FromInt32(2021)}},
			Selector: map[string]string{"name": "agent"},
		},
	}
	if err := Service(context.Background(), c, scheme, cr, desired); err != nil {
		t.Fatalf("update Service: %v", err)
	}

	updated := &corev1.Service{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "logging", Name: "agent"}, updated); err != nil {
		t.Fatalf("get Service: %v", err)
	}
	if updated.Spec.ClusterIP != "10.0.0.10" {
		t.Errorf("cluster IP = %q, want 10.0.0.10", updated.Spec.ClusterIP)
	}
	if len(updated.Spec.Ports) != 1 || updated.Spec.Ports[0].Name != "metrics" {
		t.Errorf("ports = %v, want the desired metrics port", updated.Spec.Ports)
	}
	if updated.Labels["app"] != "agent" || updated.Labels["foreign"] != "" {
		t.Errorf("labels = %v, want the desired labels only", updated.Labels)
	}
	if updated.Spec.Selector["name"] != "agent" {
		t.Errorf("selector = %v, want name=agent", updated.Spec.Selector)
	}
}

func TestStatefulSetUpdateKeepsImmutableFields(t *testing.T) {
	replicas := int32(1)
	existing := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "aggregator", Namespace: "logging"},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "aggregator-headless",
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"name": "aggregator"}},
		},
	}
	c, scheme, cr := newClient(t, existing)

	desiredReplicas := int32(3)
	desired := existing.DeepCopy()
	desired.ResourceVersion = ""
	desired.Spec.Replicas = &desiredReplicas
	desired.Spec.ServiceName = "renamed"
	if err := StatefulSet(context.Background(), c, scheme, cr, desired); err != nil {
		t.Fatalf("update StatefulSet: %v", err)
	}

	updated := &appsv1.StatefulSet{}
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "logging", Name: "aggregator"}, updated); err != nil {
		t.Fatalf("get StatefulSet: %v", err)
	}
	if *updated.Spec.Replicas != 3 {
		t.Errorf("replicas = %d, want 3", *updated.Spec.Replicas)
	}
	if updated.Spec.ServiceName != "aggregator-headless" {
		t.Errorf("service name = %q, want the immutable value kept", updated.Spec.ServiceName)
	}
}
