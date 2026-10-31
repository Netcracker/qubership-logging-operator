package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// FluentbitDefaults holds the fallback values for the FluentBit DaemonSet and Service.
// The resources and tolerations mirror charts/qubership-logging-operator/values.yaml.
type FluentbitDefaults struct {
	Resources                corev1.ResourceRequirements
	ConfigmapReloadResources corev1.ResourceRequirements
	Tolerations              []corev1.Toleration
	ConfigmapReloadArgs      []string
	HTTPPort                 int32
	LogToMetricsPort         int32
	ConfigmapReloadPort      int32
	ReloaderID               int64
	TmpVolumeSize            resource.Quantity
	MinReadySeconds          int32
	TerminationGracePeriod   int64
	MaxUnavailable           intstr.IntOrString
	LivenessProbe            corev1.Probe
	ReadinessProbe           corev1.Probe
}

func defaultFluentbit() FluentbitDefaults {
	const httpPort = 2020
	probe := func(path string) corev1.Probe {
		return corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   path,
					Port:   intstr.FromInt32(httpPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    5,
		}
	}
	return FluentbitDefaults{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
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
			{Key: "node-role.kubernetes.io/master", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		},
		ConfigmapReloadArgs: []string{
			"--volume-dir=/fluent-bit/etc",
			"--webhook-url=http://localhost:2020/api/v2/reload",
			"--webhook-retries=5",
		},
		HTTPPort:               httpPort,
		LogToMetricsPort:       2021,
		ConfigmapReloadPort:    9533,
		ReloaderID:             1001,
		TmpVolumeSize:          resource.MustParse("100Mi"),
		MinReadySeconds:        60,
		TerminationGracePeriod: 30,
		MaxUnavailable:         intstr.FromInt32(1),
		LivenessProbe:          probe("/"),
		ReadinessProbe:         probe("/api/v1/health"),
	}
}
