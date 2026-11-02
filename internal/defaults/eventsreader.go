package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EventsReaderDefaults holds the fallback values for the CloudEventsReader Deployment
// and Service. The resources mirror charts/qubership-logging-operator/values.yaml.
type EventsReaderDefaults struct {
	Command               []string
	Resources             corev1.ResourceRequirements
	Port                  int32
	RunAsID               int64
	TmpVolumeSize         resource.Quantity
	LivenessProbe         corev1.Probe
	ReadinessProbe        corev1.Probe
	RollingMaxSurge       intstr.IntOrString
	RollingMaxUnavailable intstr.IntOrString
}

func defaultEventsReader() EventsReaderDefaults {
	const port = 8080
	healthProbe := corev1.Probe{
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
	return EventsReaderDefaults{
		Command: []string{"/events-reader/eventsreader"},
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
		Port:                  port,
		RunAsID:               1000,
		TmpVolumeSize:         resource.MustParse("100Mi"),
		LivenessProbe:         healthProbe,
		ReadinessProbe:        healthProbe,
		RollingMaxSurge:       intstr.FromInt32(25),
		RollingMaxUnavailable: intstr.FromInt32(25),
	}
}
