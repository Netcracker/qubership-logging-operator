package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// EventsReaderDefaults holds default values for the CloudEventsReader component.
// Values mirror charts/qubership-logging-operator/values.yaml so deployments using
// only Helm-supplied defaults render identical Kubernetes objects after the migration.
type EventsReaderDefaults struct {
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Command         []string
	Resources       corev1.ResourceRequirements
	Port            int32
	HealthPath      string
	LivenessProbe   *corev1.Probe
	ReadinessProbe  *corev1.Probe
	SecurityContext *corev1.SecurityContext
	StrategyType    string
	MaxSurge        intstr.IntOrString
	MaxUnavailable  intstr.IntOrString
}

func defaultEventsReader() EventsReaderDefaults {
	port := int32(8080)
	healthProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/health",
				Port:   intstr.FromInt32(port),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      2,
		SuccessThreshold:    1,
		FailureThreshold:    1,
	}
	runAsNonRoot := true
	runAsUser := int64(1000)
	return EventsReaderDefaults{
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/events-reader/eventsreader"},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		Port:           port,
		HealthPath:     "/health",
		LivenessProbe:  healthProbe.DeepCopy(),
		ReadinessProbe: healthProbe.DeepCopy(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
		},
		StrategyType:   "RollingUpdate",
		MaxSurge:       intstr.FromInt32(25),
		MaxUnavailable: intstr.FromInt32(25),
	}
}
