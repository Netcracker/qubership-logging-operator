package fluentd

import (
	"strconv"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	corev1 "k8s.io/api/core/v1"
)

// buildMainEnv returns the FluentD container env: node-name field-ref, conditional
// Graylog config, queue / watch metadata defaults, MA_HOST (ipv4 vs ipv6), Ruby GC
// constants, plus per-output Loki/Http Auth env vars. Mirrors the legacy YAML asset
// 1:1; downstream callers may not assume a stable order.
func buildMainEnv(cr *loggingService.LoggingService, def config.FluentdDefaults) []corev1.EnvVar {
	spec := cr.Spec.Fluentd
	env := []corev1.EnvVar{
		{
			Name: "K8S_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			},
		},
	}
	if spec.GraylogOutput {
		env = append(env,
			corev1.EnvVar{Name: "GRAYLOG_HOST", Value: spec.GraylogHost},
			corev1.EnvVar{Name: "GRAYLOG_PORT", Value: strconv.Itoa(spec.GraylogPort)},
			corev1.EnvVar{Name: "GRAYLOG_PROTOCOL", Value: firstString(spec.GraylogProtocol, def.GraylogProtocol)},
		)
	}
	queueLen := def.QueueLimitLength
	if spec.QueueLimitLength != 0 {
		queueLen = strconv.Itoa(spec.QueueLimitLength)
	}
	env = append(env,
		corev1.EnvVar{Name: "QUEUE_LIMIT_LENGTH", Value: queueLen},
		// Legacy YAML always rendered "true" — the {{ if .WatchKubernetesMetadata }} path
		// quotes the bool which also produces "true", and the else path hard-codes "true".
		// Preserved verbatim.
		corev1.EnvVar{Name: "WATCH_KUBERNETES_METADATA", Value: def.WatchKubernetesMetadata},
		corev1.EnvVar{Name: "MA_HOST", Value: maHost(cr.Spec.Ipv6)},
	)
	env = append(env, def.RubyGCEnv...)
	if spec.Output != nil {
		if spec.Output.Loki != nil && spec.Output.Loki.Enabled {
			env = append(env, userPasswordEnv("LOKI", spec.Output.Loki.Auth)...)
		}
		if spec.Output.Http != nil && spec.Output.Http.Enabled {
			env = append(env, authEnv("HTTP", spec.Output.Http.Auth)...)
		}
	}
	return env
}

// userPasswordEnv emits prefix_USERNAME + prefix_PASSWORD if both User and Password are
// fully set. Used by Loki output, which never emits a token-as-env var in the legacy
// asset (only the token volume + mount is created — see volumes.go).
func userPasswordEnv(prefix string, auth *loggingService.Auth) []corev1.EnvVar {
	if auth == nil || !isSecretKeySet(auth.User) || !isSecretKeySet(auth.Password) {
		return nil
	}
	return []corev1.EnvVar{
		secretEnv(prefix+"_USERNAME", auth.User),
		secretEnv(prefix+"_PASSWORD", auth.Password),
	}
}

// authEnv emits USERNAME+PASSWORD when both are set, else TOKEN if set, mirroring the
// asset's else-if branch. Used by Http output.
func authEnv(prefix string, auth *loggingService.Auth) []corev1.EnvVar {
	if auth == nil {
		return nil
	}
	if isSecretKeySet(auth.User) && isSecretKeySet(auth.Password) {
		return []corev1.EnvVar{
			secretEnv(prefix+"_USERNAME", auth.User),
			secretEnv(prefix+"_PASSWORD", auth.Password),
		}
	}
	if isSecretKeySet(auth.Token) {
		return []corev1.EnvVar{secretEnv(prefix+"_TOKEN", auth.Token)}
	}
	return nil
}

func secretEnv(name string, sel *corev1.SecretKeySelector) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: sel.Name},
				Key:                  sel.Key,
			},
		},
	}
}

func maHost(ipv6 bool) string {
	if ipv6 {
		return "::"
	}
	return "0.0.0.0"
}

func isSecretKeySet(sel *corev1.SecretKeySelector) bool {
	return sel != nil && sel.Name != "" && sel.Key != ""
}

func firstString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
