package fluentbit

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// buildMainEnv returns the fluent-bit container env: the NODE_NAME field-ref plus
// per-output Auth env vars (Loki / Http / Otel). Each output emits either USERNAME +
// PASSWORD when User+Password secret refs are fully set, otherwise TOKEN when Token
// secret ref is fully set, otherwise nothing.
func buildMainEnv(spec *loggingService.Fluentbit) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			},
		},
	}
	if spec.Output == nil {
		return env
	}
	if spec.Output.Loki != nil && spec.Output.Loki.Enabled {
		env = append(env, authEnv("LOKI", spec.Output.Loki.Auth)...)
	}
	if spec.Output.Http != nil && spec.Output.Http.Enabled {
		env = append(env, authEnv("HTTP", spec.Output.Http.Auth)...)
	}
	if spec.Output.Otel != nil && spec.Output.Otel.Enabled {
		env = append(env, authEnv("OTEL", spec.Output.Otel.Auth)...)
	}
	return env
}

// authEnv returns either {prefix_USERNAME, prefix_PASSWORD} from User+Password secret
// refs, or {prefix_TOKEN} from Token secret ref, mirroring the YAML asset's else-if.
func authEnv(prefix string, auth *loggingService.Auth) []corev1.EnvVar {
	if auth == nil {
		return nil
	}
	if isSecretKeySet(auth.User) && isSecretKeySet(auth.Password) {
		return []corev1.EnvVar{
			{
				Name: prefix + "_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: auth.User.Name},
						Key:                  auth.User.Key,
					},
				},
			},
			{
				Name: prefix + "_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: auth.Password.Name},
						Key:                  auth.Password.Key,
					},
				},
			},
		}
	}
	if isSecretKeySet(auth.Token) {
		return []corev1.EnvVar{{
			Name: prefix + "_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: auth.Token.Name},
					Key:                  auth.Token.Key,
				},
			},
		}}
	}
	return nil
}

func isSecretKeySet(sel *corev1.SecretKeySelector) bool {
	return sel != nil && sel.Name != "" && sel.Key != ""
}
