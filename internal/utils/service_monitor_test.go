package utils

import (
	"testing"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGeneratePodMonitor(t *testing.T) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-metrics",
			Namespace: "logging",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "http-metrics", Port: 8383, Protocol: v1.ProtocolTCP, TargetPort: intstr.FromInt(8383)},
				{Name: "https-metrics", Port: 8443, Protocol: v1.ProtocolTCP, TargetPort: intstr.FromInt(8443)},
			},
		},
	}

	pm := GeneratePodMonitor(svc, "30s", "10s")

	if pm.Name != "test-metrics" {
		t.Errorf("expected name test-metrics, got %q", pm.Name)
	}
	if pm.Namespace != "logging" {
		t.Errorf("expected namespace logging, got %q", pm.Namespace)
	}

	if len(pm.Spec.PodMetricsEndpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(pm.Spec.PodMetricsEndpoints))
	}

	ep := pm.Spec.PodMetricsEndpoints[0]
	if *ep.Port != "http-metrics" {
		t.Errorf("expected port name http-metrics, got %q", *ep.Port)
	}
	if ep.Interval != "30s" {
		t.Errorf("expected interval 30s, got %q", ep.Interval)
	}
	if ep.ScrapeTimeout != "10s" {
		t.Errorf("expected timeout 10s, got %q", ep.ScrapeTimeout)
	}

	// Labels should contain ResourceLabels + instance label
	if pm.Labels["app.kubernetes.io/name"] != "test-metrics" {
		t.Error("missing app.kubernetes.io/name label")
	}
	if pm.Labels["app.kubernetes.io/component"] != "monitoring" {
		t.Error("missing component label")
	}
	if _, ok := pm.Labels["app.kubernetes.io/instance"]; !ok {
		t.Error("missing instance label")
	}
}

func TestPopulateEndpointsFromServicePorts(t *testing.T) {
	t.Run("empty ports", func(t *testing.T) {
		svc := &v1.Service{Spec: v1.ServiceSpec{}}
		endpoints := populateEndpointsFromServicePorts(svc, "30s", "10s")
		if len(endpoints) != 0 {
			t.Errorf("expected 0 endpoints, got %d", len(endpoints))
		}
	})

	t.Run("single port", func(t *testing.T) {
		svc := &v1.Service{
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{Name: "metrics", Port: 8080},
				},
			},
		}
		endpoints := populateEndpointsFromServicePorts(svc, "15s", "5s")
		if len(endpoints) != 1 {
			t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
		}
		if *endpoints[0].Port != "metrics" {
			t.Errorf("expected port name metrics, got %q", *endpoints[0].Port)
		}
		if endpoints[0].Interval != promv1.Duration("15s") {
			t.Errorf("expected interval 15s, got %q", endpoints[0].Interval)
		}
	})
}
