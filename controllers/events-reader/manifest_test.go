package events_reader

import (
	"reflect"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEventsReaderDeploymentUsesHardenedSecurityContext(t *testing.T) {
	deployment, err := eventsReaderDeployment(newEventsReaderLoggingService(false))
	if err != nil {
		t.Fatalf("render Events Reader Deployment: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.RunAsNonRoot == nil ||
		!*podSpec.SecurityContext.RunAsNonRoot || podSpec.SecurityContext.RunAsUser == nil ||
		*podSpec.SecurityContext.RunAsUser != 1000 || podSpec.SecurityContext.RunAsGroup == nil ||
		*podSpec.SecurityContext.RunAsGroup != 1000 || podSpec.SecurityContext.SeccompProfile == nil ||
		podSpec.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Fatalf("unexpected pod security context: %#v", podSpec.SecurityContext)
	}

	container := podSpec.Containers[0]
	securityContext := container.SecurityContext
	if securityContext == nil || securityContext.AllowPrivilegeEscalation == nil ||
		*securityContext.AllowPrivilegeEscalation || securityContext.ReadOnlyRootFilesystem == nil ||
		!*securityContext.ReadOnlyRootFilesystem || securityContext.RunAsNonRoot == nil ||
		!*securityContext.RunAsNonRoot || securityContext.RunAsUser == nil ||
		*securityContext.RunAsUser != 1000 || securityContext.RunAsGroup == nil ||
		*securityContext.RunAsGroup != 1000 {
		t.Fatalf("unexpected container security context: %#v", securityContext)
	}
	if securityContext.Capabilities == nil ||
		!reflect.DeepEqual(securityContext.Capabilities.Drop, []corev1.Capability{"ALL"}) {
		t.Fatalf("unexpected container capabilities: %#v", securityContext.Capabilities)
	}

	assertEventsReaderTmpVolume(t, podSpec.Volumes, container.VolumeMounts)
}

func TestEventsReaderDeploymentLetsOpenShiftAssignTheUser(t *testing.T) {
	deployment, err := eventsReaderDeployment(newEventsReaderLoggingService(true))
	if err != nil {
		t.Fatalf("render Events Reader Deployment: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.RunAsNonRoot == nil ||
		!*podSpec.SecurityContext.RunAsNonRoot || podSpec.SecurityContext.RunAsUser != nil ||
		podSpec.SecurityContext.RunAsGroup == nil || *podSpec.SecurityContext.RunAsGroup != 1000 {
		t.Fatalf("unexpected OpenShift pod security context: %#v", podSpec.SecurityContext)
	}
	securityContext := podSpec.Containers[0].SecurityContext
	if securityContext == nil || securityContext.RunAsNonRoot == nil ||
		!*securityContext.RunAsNonRoot || securityContext.RunAsUser != nil || securityContext.RunAsGroup == nil ||
		*securityContext.RunAsGroup != 1000 {
		t.Fatalf("unexpected OpenShift container security context: %#v", securityContext)
	}
}

func assertEventsReaderTmpVolume(t *testing.T, volumes []corev1.Volume, mounts []corev1.VolumeMount) {
	t.Helper()
	expectedLimit := resource.MustParse("100Mi")
	for _, volume := range volumes {
		if volume.Name == "tmp" && volume.EmptyDir != nil && volume.EmptyDir.SizeLimit != nil &&
			volume.EmptyDir.SizeLimit.Cmp(expectedLimit) == 0 {
			for _, mount := range mounts {
				if mount.Name == "tmp" && mount.MountPath == "/tmp" {
					return
				}
			}
		}
	}
	t.Fatal("Events Reader must mount a 100Mi emptyDir at /tmp")
}

func newEventsReaderLoggingService(openshift bool) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{Name: "logging-service", Namespace: "logging"},
		Spec: loggingService.LoggingServiceSpec{
			CloudURL:        "https://kubernetes.default.svc",
			OpenshiftDeploy: openshift,
			CloudEventsReader: &loggingService.CloudEventsReader{
				DockerImage: "ghcr.io/netcracker/qubership-kube-events-reader:main",
			},
		},
	}
}
