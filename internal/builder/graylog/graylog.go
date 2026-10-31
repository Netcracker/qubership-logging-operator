// Package graylog builds the Graylog StatefulSet, Service, and ServiceAccount from the
// LoggingService custom resource. The callers check that cr.Spec.Graylog is set.
package graylog

import (
	"fmt"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	statefulSetName    = "graylog"
	serviceName        = "graylog-service"
	serviceAccountName = "logging-graylog"
	component          = "graylog"
	technology         = "java-others"

	httpPort         = 9000
	udpPort          = 514
	metricsPort      = 9833
	mongoPort        = 27017
	authProxyPort    = 8888
	authProxyMetrics = 8889
	authProxyName    = "graylog-auth-proxy"
)

// selector links the Service and the StatefulSet to the Graylog pods.
func selector() map[string]string {
	return map[string]string{"name": statefulSetName}
}

// Service returns the Graylog Service. With the auth proxy installed, port 9000 leads
// to the proxy, and the Service also exposes the proxy metrics.
func Service(cr *loggingService.LoggingService, _ *defaults.Defaults) *corev1.Service {
	spec := cr.Spec.Graylog
	var ports []corev1.ServicePort
	if authProxyInstalled(spec) {
		ports = append(ports,
			servicePort("graylog", httpPort, authProxyPort, corev1.ProtocolTCP),
			servicePort("metrics", authProxyMetrics, authProxyMetrics, corev1.ProtocolTCP))
	} else {
		ports = append(ports, servicePort("graylog", httpPort, httpPort, corev1.ProtocolTCP))
	}
	inputPort := int32(spec.InputPort)
	ports = append(ports,
		servicePort("graylog-udp", udpPort, udpPort, corev1.ProtocolUDP),
		servicePort(fmt.Sprintf("graylog-%d", inputPort), inputPort, inputPort, corev1.ProtocolTCP),
		servicePort("graylog-metrics", metricsPort, metricsPort, corev1.ProtocolTCP))

	service := builder.NewService(serviceName, cr.GetNamespace(), builder.ServiceOpts{
		Selector: selector(),
		Ports:    ports,
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            serviceName,
		Component:       component,
		ComponentLabels: spec.Labels,
	}, nil)
	return service
}

// ServiceAccount returns the ServiceAccount the Graylog pods run with.
func ServiceAccount(cr *loggingService.LoggingService, _ *defaults.Defaults) *corev1.ServiceAccount {
	serviceAccount := builder.NewServiceAccount(serviceAccountName, cr.GetNamespace())
	util.SetLabelsForResource(serviceAccount, util.LabelInput{
		Name:            serviceAccountName,
		Component:       component,
		ComponentLabels: cr.Spec.Graylog.Labels,
	}, nil)
	return serviceAccount
}

func servicePort(name string, port, targetPort int32, protocol corev1.Protocol) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       name,
		Port:       port,
		Protocol:   protocol,
		TargetPort: intstr.FromInt32(targetPort),
	}
}

func authProxyInstalled(spec *loggingService.Graylog) bool {
	return spec.AuthProxy != nil && spec.AuthProxy.Install
}

func httpTLSEnabled(spec *loggingService.Graylog) bool {
	return spec.TLS != nil && spec.TLS.HTTP != nil && spec.TLS.HTTP.Enabled
}
