package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// FluentbitAggregatorDefaults holds defaults for the FluentBit HA pair (forwarder
// DaemonSet + aggregator StatefulSet). Values mirror
// charts/qubership-logging-operator/values.yaml and the legacy YAML assets at
// controllers/fluentbit-forwarder-aggregator/assets/.
type FluentbitAggregatorDefaults struct {
	// Forwarder-specific defaults.
	ForwarderLivenessProbe  *corev1.Probe
	ForwarderReadinessProbe *corev1.Probe
	ForwarderMaxUnavailable intstr.IntOrString
	ForwarderMinReadyS      int32
	ForwarderTGracePeriodS  int64

	// Aggregator-specific defaults.
	AggregatorReplicas            int32
	AggregatorResources           corev1.ResourceRequirements
	AggregatorConfigmapReloadRes  corev1.ResourceRequirements
	AggregatorLivenessProbe       *corev1.Probe
	AggregatorReadinessProbe      *corev1.Probe
	AggregatorTGracePeriodS       int64
	AggregatorRevisionHistory     int32
	AggregatorPodSecurityContext  *corev1.PodSecurityContext
	AggregatorContainerSecCtx     *corev1.SecurityContext
	AggregatorPVCStorageSize      string

	// Shared ports between forwarder and aggregator main containers.
	HTTPPort            int32
	LogToMetricsPort    int32
	ForwardPort         int32
	ConfigmapReloadPort int32

	// Shared sidecar argv.
	ConfigmapReloadArgs []string
}

func defaultAggregator() FluentbitAggregatorDefaults {
	httpPort := int32(2020)
	logToMetricsPort := int32(2021)
	forwardPort := int32(24224)
	reloadPort := int32(9533)

	probe := func(path string, failure int32) *corev1.Probe {
		return &corev1.Probe{
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
			FailureThreshold:    failure,
		}
	}

	runAsUser := int64(0)
	runAsNonRoot := false
	fsGroup := int64(1001)
	fsGroupPolicy := corev1.FSGroupChangeOnRootMismatch
	revisionHistory := int32(10)

	return FluentbitAggregatorDefaults{
		ForwarderLivenessProbe:  probe("/", 3),
		ForwarderReadinessProbe: probe("/api/v1/health", 5),
		ForwarderMaxUnavailable: intstr.FromInt32(1),
		ForwarderMinReadyS:      60,
		ForwarderTGracePeriodS:  30,

		AggregatorReplicas: 2,
		AggregatorResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		},
		AggregatorConfigmapReloadRes: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("10Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		},
		AggregatorLivenessProbe:  probe("/", 5),
		AggregatorReadinessProbe: probe("/api/v1/health", 5),
		AggregatorTGracePeriodS:  30,
		AggregatorRevisionHistory: revisionHistory,
		AggregatorPodSecurityContext: &corev1.PodSecurityContext{
			RunAsUser:           &runAsUser,
			RunAsNonRoot:        &runAsNonRoot,
			FSGroup:             &fsGroup,
			FSGroupChangePolicy: &fsGroupPolicy,
		},
		AggregatorContainerSecCtx: &corev1.SecurityContext{
			RunAsUser:    &runAsUser,
			RunAsNonRoot: &runAsNonRoot,
		},
		AggregatorPVCStorageSize: "2Gi",

		HTTPPort:            httpPort,
		LogToMetricsPort:    logToMetricsPort,
		ForwardPort:         forwardPort,
		ConfigmapReloadPort: reloadPort,
		ConfigmapReloadArgs: []string{
			"--volume-dir=/fluent-bit/etc",
			"--webhook-url=http://localhost:2020/api/v2/reload",
			"--webhook-retries=5",
		},
	}
}
