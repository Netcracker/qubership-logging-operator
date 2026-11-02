package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// FluentdDefaults holds the fallback values for the FluentD DaemonSet and Service.
// The resources and tolerations mirror charts/qubership-logging-operator/values.yaml.
type FluentdDefaults struct {
	Resources                corev1.ResourceRequirements
	ConfigmapReloadResources corev1.ResourceRequirements
	Tolerations              []corev1.Toleration
	HTTPPort                 int32
	MetricsPort              int32
	ConfigmapReloadPort      int32
	LivenessProbe            corev1.Probe
	ReadinessProbe           corev1.Probe
	GraylogProtocol          string
	QueueLimitLength         int
	TmpVolumeSize            resource.Quantity
}

func defaultFluentd() FluentdDefaults {
	const httpPort = 24220
	probe := func(initialDelay, failureThreshold int32) corev1.Probe {
		return corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/api/plugins.json",
					Port:   intstr.FromInt32(httpPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: initialDelay,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    failureThreshold,
		}
	}
	return FluentdDefaults{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
		ConfigmapReloadResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("10Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		},
		Tolerations: []corev1.Toleration{
			{Key: "node-role.kubernetes.io/master", Operator: corev1.TolerationOpExists},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		},
		HTTPPort:            httpPort,
		MetricsPort:         24232,
		ConfigmapReloadPort: 9533,
		LivenessProbe:       probe(90, 3),
		ReadinessProbe:      probe(10, 5),
		GraylogProtocol:     "tcp",
		QueueLimitLength:    5000,
		TmpVolumeSize:       resource.MustParse("100Mi"),
	}
}
