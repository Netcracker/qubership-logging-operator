package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

// EnsureSecretRootPasswordSHA2 sets secret.Data[util.GraylogSecretKeyRootPasswordSHA2] to SHA256hex(plainPassword)
// when missing or different. Returns true if the secret was modified (caller must persist with Update).
func EnsureSecretRootPasswordSHA2(secret *corev1.Secret, plainPassword string) bool {
	sum := sha256.Sum256([]byte(plainPassword))
	want := hex.EncodeToString(sum[:])
	key := util.GraylogSecretKeyRootPasswordSHA2
	got := ""
	if secret.Data != nil && secret.Data[key] != nil {
		got = strings.TrimSpace(string(secret.Data[key]))
	}
	if got == want {
		return false
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[key] = []byte(want)
	return true
}
