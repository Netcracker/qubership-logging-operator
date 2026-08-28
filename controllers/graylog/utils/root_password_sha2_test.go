package utils

import (
	"testing"

	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

const adminSHA256 = "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"

func TestEnsureSecretRootPasswordSHA2(t *testing.T) {
	tests := []struct {
		name      string
		secret    *corev1.Secret
		wantDirty bool
	}{
		{
			name:      "sets hash when data is nil",
			secret:    &corev1.Secret{},
			wantDirty: true,
		},
		{
			name: "sets hash when key is missing",
			secret: &corev1.Secret{
				Data: map[string][]byte{},
			},
			wantDirty: true,
		},
		{
			name: "does not change matching hash",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					util.GraylogSecretKeyRootPasswordSHA2: []byte(adminSHA256),
				},
			},
			wantDirty: false,
		},
		{
			name: "trims existing hash before comparing",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					util.GraylogSecretKeyRootPasswordSHA2: []byte(adminSHA256 + "\n"),
				},
			},
			wantDirty: false,
		},
		{
			name: "replaces different hash",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					util.GraylogSecretKeyRootPasswordSHA2: []byte("old"),
				},
			},
			wantDirty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirty := EnsureSecretRootPasswordSHA2(tt.secret, "admin")
			if dirty != tt.wantDirty {
				t.Fatalf("EnsureSecretRootPasswordSHA2() dirty = %v, want %v", dirty, tt.wantDirty)
			}
			got := string(tt.secret.Data[util.GraylogSecretKeyRootPasswordSHA2])
			if got != adminSHA256 && tt.wantDirty {
				t.Fatalf("hash = %q, want %q", got, adminSHA256)
			}
		})
	}
}
