package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// FluentdDefaults holds default values for the FluentD DaemonSet + Service. Values
// mirror charts/qubership-logging-operator/values.yaml and the historical YAML asset
// at controllers/fluentd/assets/daemon-set.yaml so existing deployments render
// identical Kubernetes objects after the migration.
type FluentdDefaults struct {
	Resources                     corev1.ResourceRequirements
	ConfigmapReloadResources      corev1.ResourceRequirements
	Tolerations                   []corev1.Toleration
	MinReadySeconds               int32
	TerminationGracePeriodSeconds int64
	MaxUnavailable                intstr.IntOrString
	HTTPPort                      int32
	MetricsPort                   int32
	ConfigmapReloadPort           int32
	LivenessProbe                 *corev1.Probe
	ReadinessProbe                *corev1.Probe
	ConfigmapReloadArgs           []string
	GraylogProtocol               string
	QueueLimitLength              string
	WatchKubernetesMetadata       string
	RubyGCEnv                     []corev1.EnvVar
}

func defaultFluentd() FluentdDefaults {
	httpPort := int32(24220)
	metricsPort := int32(24232)
	configmapReloadPort := int32(9533)
	probeAction := corev1.HTTPGetAction{
		Path:   "/api/plugins.json",
		Port:   intstr.FromInt32(httpPort),
		Scheme: corev1.URISchemeHTTP,
	}
	liveness := &corev1.Probe{
		ProbeHandler:        corev1.ProbeHandler{HTTPGet: probeAction.DeepCopy()},
		InitialDelaySeconds: 90,
		TimeoutSeconds:      10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readiness := &corev1.Probe{
		ProbeHandler:        corev1.ProbeHandler{HTTPGet: probeAction.DeepCopy()},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    5,
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
		MinReadySeconds:               60,
		TerminationGracePeriodSeconds: 30,
		MaxUnavailable:                intstr.FromInt32(1),
		HTTPPort:                      httpPort,
		MetricsPort:                   metricsPort,
		ConfigmapReloadPort:           configmapReloadPort,
		LivenessProbe:                 liveness,
		ReadinessProbe:                readiness,
		ConfigmapReloadArgs: []string{
			"--volume-dir=/fluentd/etc",
			"--webhook-url=http://localhost:24444/api/config.reload",
		},
		GraylogProtocol:         "tcp",
		QueueLimitLength:        "5000",
		WatchKubernetesMetadata: "true",
		RubyGCEnv: []corev1.EnvVar{
			{Name: "RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR", Value: "1.2"},
			{Name: "RUBY_GC_MALLOC_LIMIT", Value: "4194304"},
			{Name: "RUBY_GC_MALLOC_LIMIT_MAX", Value: "16777216"},
			{Name: "RUBY_GC_OLDMALLOC_LIMIT", Value: "16777216"},
			{Name: "RUBY_GC_OLDMALLOC_LIMIT_MAX", Value: "16777216"},
		},
	}
}
