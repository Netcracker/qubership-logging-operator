// Package graylog builds the Graylog ServiceAccount, StatefulSet, Service, and the
// MongoDB upgrade Jobs from the LoggingService CR plus code-level defaults. Replaces
// the seven YAML assets under controllers/graylog/assets/. The ConfigMap path stays
// in controllers/graylog/manifest.go.
package graylog

import (
	"context"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/reconcile"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// StatefulSetName is the Graylog StatefulSet name (and the pod-template "name" label).
	StatefulSetName = "graylog"
	// ServiceAccountName is the Graylog ServiceAccount name.
	ServiceAccountName = "logging-graylog"
	// ServiceName is the Graylog Service name. Matches util.GraylogComponentName.
	ServiceName = "graylog-service"
	// GraylogContainer is the main Graylog container name.
	GraylogContainer = "graylog"
	// MongoContainer is the embedded MongoDB sidecar name.
	MongoContainer = "mongo"
	// AuthProxyContainer is the optional auth-proxy sidecar name.
	AuthProxyContainer = "graylog-auth-proxy"
	// SetupInit is the init container that prepares /usr/share/graylog/data.
	SetupInit = "setup"
	// DownloadPluginsInit is the optional init container that downloads plugins.
	DownloadPluginsInit = "download-plugins"
	// Technology is set on the pod-template app.kubernetes.io/technology label.
	Technology = "java-others"

	// GraylogClaim is the data PVC claim name. Matches util.GraylogClaimName.
	GraylogClaim = "graylog-claim"
	// MongoClaim is the MongoDB PVC claim name. Matches util.MongoClaimName.
	MongoClaim = "mongo-claim"
)

// CreateOrUpdate reconciles ServiceAccount + StatefulSet + Service. The MongoDB
// upgrade Jobs are reconciled separately by the controller in its sequential
// orchestration loop (see BuildMongoUpgradeJob).
func CreateOrUpdate(ctx context.Context, c client.Client, scheme *runtime.Scheme, cr *loggingService.LoggingService, cfg *config.Defaults) error {
	if err := reconcile.ServiceAccount(ctx, c, scheme, cr, buildServiceAccount(cr)); err != nil {
		return err
	}
	if err := reconcile.StatefulSet(ctx, c, scheme, cr, buildStatefulSet(cr, cfg.Graylog)); err != nil {
		return err
	}
	return reconcile.Service(ctx, c, scheme, cr, buildService(cr, cfg.Graylog))
}

func buildServiceAccount(cr *loggingService.LoggingService) *corev1.ServiceAccount {
	sa := build.NewServiceAccount(ServiceAccountName, cr.GetNamespace(), "graylog")
	util.SetLabelsForResource(sa, util.LabelInput{
		Name:            ServiceAccountName,
		Component:       "graylog",
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return sa
}

func buildService(cr *loggingService.LoggingService, def config.GraylogDefaults) *corev1.Service {
	spec := cr.Spec.Graylog
	httpPort := def.HTTPPort
	udpPort := def.UDPPort
	metricsPort := def.MetricsPort
	inputPort := int32(spec.InputPort)

	ports := graylogServicePorts(spec, def, httpPort)
	ports = append(ports,
		corev1.ServicePort{Name: "graylog-udp", Port: udpPort, TargetPort: intstr.FromInt32(udpPort), Protocol: corev1.ProtocolUDP},
		corev1.ServicePort{Name: graylogInputPortName(inputPort), Port: inputPort, TargetPort: intstr.FromInt32(inputPort), Protocol: corev1.ProtocolTCP},
		corev1.ServicePort{Name: "graylog-metrics", Port: metricsPort, TargetPort: intstr.FromInt32(metricsPort), Protocol: corev1.ProtocolTCP},
	)
	svc := build.NewService(ServiceName, cr.GetNamespace(), "graylog", build.ServiceOpts{
		Type:     corev1.ServiceTypeClusterIP,
		Ports:    ports,
		Selector: map[string]string{"name": StatefulSetName},
	})
	util.SetLabelsForResource(svc, util.LabelInput{
		Name:            ServiceName,
		Component:       "graylog",
		ComponentLabels: spec.Labels,
	}, nil)
	return svc
}

// graylogServicePorts returns the first two service ports. When AuthProxy is installed
// the proxy fronts Graylog (port 9000 → proxy 8888) and exposes a separate metrics
// port; otherwise port 9000 points straight at the Graylog container.
func graylogServicePorts(spec *loggingService.Graylog, def config.GraylogDefaults, httpPort int32) []corev1.ServicePort {
	if spec.AuthProxy != nil && spec.AuthProxy.Install {
		return []corev1.ServicePort{
			{Name: "graylog", Port: httpPort, TargetPort: intstr.FromInt32(def.AuthProxyHTTPPort), Protocol: corev1.ProtocolTCP},
			{Name: "metrics", Port: def.AuthProxyMetricsPort, TargetPort: intstr.FromInt32(def.AuthProxyMetricsPort), Protocol: corev1.ProtocolTCP},
		}
	}
	return []corev1.ServicePort{
		{Name: "graylog", Port: httpPort, TargetPort: intstr.FromInt32(httpPort), Protocol: corev1.ProtocolTCP},
	}
}

// graylogInputPortName mirrors the asset's `graylog-<InputPort>` port-name convention.
func graylogInputPortName(p int32) string {
	return "graylog-" + itoa(p)
}

// itoa renders a small int32 as decimal without pulling strconv into multiple files.
func itoa(i int32) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [12]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func tlsHTTPEnabled(g *loggingService.Graylog) bool {
	return g.TLS != nil && g.TLS.HTTP != nil
}

func tlsHTTPListenScheme(g *loggingService.Graylog) corev1.URIScheme {
	if g.TLS != nil && g.TLS.HTTP != nil && g.TLS.HTTP.Enabled {
		return corev1.URISchemeHTTPS
	}
	return corev1.URISchemeHTTP
}
