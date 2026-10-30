package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// FluentbitDefaults holds default values for the FluentBit DaemonSet + Service.
// Values mirror charts/qubership-logging-operator/values.yaml and the historical YAML
// asset at controllers/fluentbit/assets/daemon-set.yaml so existing deployments render
// identical Kubernetes objects after the migration.
type FluentbitDefaults struct {
	Resources                     corev1.ResourceRequirements
	ConfigmapReloadResources      corev1.ResourceRequirements
	Tolerations                   []corev1.Toleration
	MinReadySeconds               int32
	TerminationGracePeriodSeconds int64
	MaxUnavailable                intstr.IntOrString
	HTTPPort                      int32
	LogToMetricsPort              int32
	ConfigmapReloadPort           int32
	LivenessProbe                 *corev1.Probe
	ReadinessProbe                *corev1.Probe
	ConfigmapReloadArgs           []string
}

func defaultFluentbit() FluentbitDefaults {
	httpPort := int32(2020)
	logToMetricsPort := int32(2021)
	configmapReloadPort := int32(9533)
	liveness := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/",
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
	readiness := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/api/v1/health",
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
			{Key: "node-role.kubernetes.io/master", Operator: corev1.TolerationOpExists},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
			{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		},
		MinReadySeconds:               60,
		TerminationGracePeriodSeconds: 30,
		MaxUnavailable:                intstr.FromInt32(1),
		HTTPPort:                      httpPort,
		LogToMetricsPort:              logToMetricsPort,
		ConfigmapReloadPort:           configmapReloadPort,
		LivenessProbe:                 liveness,
		ReadinessProbe:                readiness,
		ConfigmapReloadArgs: []string{
			"--volume-dir=/fluent-bit/etc",
			"--webhook-url=http://localhost:2020/api/v2/reload",
			"--webhook-retries=5",
		},
	}
}
