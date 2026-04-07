package graylog

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
)

func TestCheckGraylog5(t *testing.T) {
	r := &GraylogReconciler{
		ComponentReconciler: &util.ComponentReconciler{
			Log: util.Logger("test-graylog"),
		},
	}

	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{"graylog 5.2.1", "graylog/graylog:5.2.1", true},
		{"graylog 4.3.0", "graylog/graylog:4.3.0", false},
		{"graylog 6.0.0", "graylog/graylog:6.0.0", false},
		{"registry with port", "registry:5000/graylog:5.0.0-rc1", true},
		{"latest tag no semver", "graylog/graylog:latest", false},
		{"no tag at all", "graylog", false},
		{"version 5.0.0.1 extra segments", "graylog:5.0.0.1", true},
		{"empty image", "", false},
		{"just version 5.0.0", "5.0.0", true},
		// Note: regex matches first semver in the entire string, not just the tag.
		// This is acceptable since real Docker images don't have semver in the path.
		{"version in path picks first semver", "5.0.0/graylog:4.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &loggingService.LoggingService{
				Spec: loggingService.LoggingServiceSpec{
					Graylog: &loggingService.Graylog{
						DockerImage: tt.image,
					},
				},
			}
			result := r.checkGraylog5(cr)
			if result != tt.expected {
				t.Errorf("checkGraylog5(%q) = %v, want %v", tt.image, result, tt.expected)
			}
		})
	}
}
