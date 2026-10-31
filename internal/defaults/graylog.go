package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GraylogDefaults holds the fallback values for the Graylog StatefulSet, Service, and
// ServiceAccount. The values mirror charts/qubership-logging-operator/values.yaml.
type GraylogDefaults struct {
	Replicas           int32
	GraylogResources   corev1.ResourceRequirements
	MongoResources     corev1.ResourceRequirements
	InitResources      corev1.ResourceRequirements
	AuthProxyResources corev1.ResourceRequirements
	JavaOpts           string
	EmptyDirSize       resource.Quantity
}

func defaultGraylog() GraylogDefaults {
	return GraylogDefaults{
		Replicas:           1,
		GraylogResources:   graylogResourceRequirements("500m", "1536Mi", "1000m", "2048Mi"),
		MongoResources:     graylogResourceRequirements("500m", "256Mi", "500m", "256Mi"),
		InitResources:      graylogResourceRequirements("50m", "128Mi", "100m", "256Mi"),
		AuthProxyResources: graylogResourceRequirements("100m", "128Mi", "200m", "512Mi"),
		JavaOpts:           "-Xms1024m -Xmx1024m -Djna.tmpdir=/usr/share/graylog/data/plugin",
		EmptyDirSize:       resource.MustParse("100Mi"),
	}
}

func graylogResourceRequirements(requestCPU, requestMemory, limitCPU, limitMemory string) corev1.ResourceRequirements {
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
