package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// GraylogDefaults holds default values for the Graylog StatefulSet, Service,
// ServiceAccount, and the MongoDB upgrade Jobs. Values mirror
// charts/qubership-logging-operator/values.yaml and the historical YAML assets at
// controllers/graylog/assets/.
type GraylogDefaults struct {
	Replicas               int32
	GraylogResources       corev1.ResourceRequirements
	MongoResources         corev1.ResourceRequirements
	InitResources          corev1.ResourceRequirements
	JavaOpts               string
	PathRepo               string
	LivenessProbe          *corev1.Probe
	ReadinessProbe         *corev1.Probe
	HTTPPort               int32
	UDPPort                int32
	MetricsPort            int32
	AuthProxyHTTPPort      int32
	AuthProxyMetricsPort   int32
	MongoPort              int32
	PodSecurityContext     *corev1.PodSecurityContext
	UpgradeJobBackoffLimit int32
	MaxSurge               intstr.IntOrString
	MaxUnavailable         intstr.IntOrString
}

func defaultGraylog() GraylogDefaults {
	runAsNonRoot := false
	runAsUser := int64(0)
	fsGroup := int64(1001)
	fsGroupPolicy := corev1.FSGroupChangeOnRootMismatch

	probe := func(scheme corev1.URIScheme) *corev1.Probe {
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt32(9000),
					Scheme: scheme,
				},
			},
			InitialDelaySeconds: 120,
			PeriodSeconds:       30,
			TimeoutSeconds:      5,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}
	}

	return GraylogDefaults{
		Replicas: 1,
		GraylogResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1536Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2048Mi"),
			},
		},
		MongoResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		InitResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		JavaOpts:             "-Xms1024m -Xmx1024m -Djna.tmpdir=/usr/share/graylog/data/plugin",
		PathRepo:             "/usr/share/opensearch/snapshots/graylog/",
		LivenessProbe:        probe(corev1.URISchemeHTTP),
		ReadinessProbe:       probe(corev1.URISchemeHTTP),
		HTTPPort:             9000,
		UDPPort:              514,
		MetricsPort:          9833,
		AuthProxyHTTPPort:    8888,
		AuthProxyMetricsPort: 8889,
		MongoPort:            27017,
		PodSecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot:        &runAsNonRoot,
			RunAsUser:           &runAsUser,
			FSGroup:             &fsGroup,
			FSGroupChangePolicy: &fsGroupPolicy,
		},
		UpgradeJobBackoffLimit: 3,
		MaxSurge:               intstr.FromInt32(25),
		MaxUnavailable:         intstr.FromInt32(25),
	}
}
