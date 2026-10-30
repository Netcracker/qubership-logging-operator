package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// FluentbitAggregatorDefaults holds the fallback values for the FluentBit HA mode: the
// forwarder DaemonSet and the aggregator StatefulSet. The values mirror the fluentbit
// and fluentbit.aggregator sections of charts/qubership-logging-operator/values.yaml.
type FluentbitAggregatorDefaults struct {
	// ForwarderResources applies when fluentbit.resources is not set.
	ForwarderResources corev1.ResourceRequirements
	// AggregatorResources applies when fluentbit.aggregator.resources is not set.
	AggregatorResources corev1.ResourceRequirements
	// ConfigmapReloadResources applies to the configmap-reload sidecar of both workloads
	// when its resources are not set.
	ConfigmapReloadResources corev1.ResourceRequirements
	// AggregatorReplicas applies when fluentbit.aggregator.replicas is not set.
	AggregatorReplicas int32
	// AggregatorVolumeSize applies when fluentbit.aggregator.volume.bind is set without
	// a storage size.
	AggregatorVolumeSize resource.Quantity
}

func defaultAggregator() FluentbitAggregatorDefaults {
	return FluentbitAggregatorDefaults{
		ForwarderResources:       haResources("50m", "128Mi", "200m", "512Mi"),
		AggregatorResources:      haResources("500m", "512Mi", "2", "2Gi"),
		ConfigmapReloadResources: haResources("10m", "10Mi", "50m", "50Mi"),
		AggregatorReplicas:       2,
		AggregatorVolumeSize:     resource.MustParse("2Gi"),
	}
}

// haResources returns CPU and memory requests and limits parsed from the given quantities.
func haResources(requestCPU, requestMemory, limitCPU, limitMemory string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(requestCPU),
			corev1.ResourceMemory: resource.MustParse(requestMemory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(limitCPU),
			corev1.ResourceMemory: resource.MustParse(limitMemory),
		},
	}
}
