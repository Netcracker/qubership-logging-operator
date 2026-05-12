package forwarderaggregator

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// dockerHostPathVolumes mirrors the asset's docker-runtime volumes (sysconfig skipped on Ubuntu).
func dockerHostPathVolumes(runtime, osKind string) []corev1.Volume {
	if runtime != containerRuntimeDocker {
		return nil
	}
	fileOrCreate := corev1.HostPathFileOrCreate
	var vols []corev1.Volume
	if osKind != osKindUbuntu {
		vols = append(vols, corev1.Volume{
			Name: "sysconfigdocker",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/etc/sysconfig/docker", Type: &fileOrCreate},
			},
		})
	}
	vols = append(vols,
		corev1.Volume{Name: "dockerdaemon", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/docker/daemon.json", Type: &fileOrCreate}}},
		corev1.Volume{Name: "dockerhostname", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/hostname", Type: &fileOrCreate}}},
		corev1.Volume{Name: "varlibdockercontainers", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/docker/containers", Type: hostPathType("")}}},
	)
	return vols
}

func dockerVolumeMounts(runtime, osKind string) []corev1.VolumeMount {
	if runtime != containerRuntimeDocker {
		return nil
	}
	var mounts []corev1.VolumeMount
	if osKind != osKindUbuntu {
		mounts = append(mounts, corev1.VolumeMount{Name: "sysconfigdocker", MountPath: "/etc/sysconfig/docker", ReadOnly: true})
	}
	mounts = append(mounts,
		corev1.VolumeMount{Name: "dockerdaemon", MountPath: "/etc/docker/daemon.json", ReadOnly: true},
		corev1.VolumeMount{Name: "dockerhostname", MountPath: "/etc/docker-hostname", ReadOnly: true},
		corev1.VolumeMount{Name: "varlibdockercontainers", MountPath: "/var/lib/docker/containers", ReadOnly: true},
	)
	return mounts
}

// tlsVolumes returns secret-backed volumes for the main TLS material. When GenerateCerts
// is enabled, the three logical names share a single secret; otherwise each maps to its
// CA/Cert/Key SecretName from the CR.
func tlsVolumes(tls *loggingService.FluentbitTLS) []corev1.Volume {
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		name := tls.GenerateCerts.SecretName
		return []corev1.Volume{
			secretVolume("tls-ca", name, configMapDefaultMode),
			secretVolume("tls-cert", name, configMapDefaultMode),
			secretVolume("tls-key", name, configMapDefaultMode),
		}
	}
	var vols []corev1.Volume
	if isCASet(tls.CA) {
		vols = append(vols, secretVolume("tls-ca", tls.CA.SecretName, configMapDefaultMode))
	}
	if isCertSet(tls.Cert) {
		vols = append(vols, secretVolume("tls-cert", tls.Cert.SecretName, configMapDefaultMode))
	}
	if isKeySet(tls.Key) {
		vols = append(vols, secretVolume("tls-key", tls.Key.SecretName, configMapDefaultMode))
	}
	return vols
}

func tlsVolumeMounts(tls *loggingService.FluentbitTLS) []corev1.VolumeMount {
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		return []corev1.VolumeMount{
			{Name: "tls-ca", MountPath: "/fluent-bit/tls/ca.crt", ReadOnly: true, SubPath: "ca.crt"},
			{Name: "tls-cert", MountPath: "/fluent-bit/tls/tls.crt", ReadOnly: true, SubPath: "tls.crt"},
			{Name: "tls-key", MountPath: "/fluent-bit/tls/tls.key", ReadOnly: true, SubPath: "tls.key"},
		}
	}
	var mounts []corev1.VolumeMount
	if isCASet(tls.CA) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-ca", MountPath: "/fluent-bit/tls/ca.crt", ReadOnly: true, SubPath: tls.CA.SecretKey})
	}
	if isCertSet(tls.Cert) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-cert", MountPath: "/fluent-bit/tls/tls.crt", ReadOnly: true, SubPath: tls.Cert.SecretKey})
	}
	if isKeySet(tls.Key) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-key", MountPath: "/fluent-bit/tls/tls.key", ReadOnly: true, SubPath: tls.Key.SecretKey})
	}
	return mounts
}

func outputTLSVolumes(prefix string, certs loggingService.Certificates) []corev1.Volume {
	var vols []corev1.Volume
	if isCASet(certs.CA) {
		vols = append(vols, secretVolume(prefix+"-tls-ca", certs.CA.SecretName, configMapDefaultMode))
	}
	if isCertSet(certs.Cert) {
		vols = append(vols, secretVolume(prefix+"-tls-cert", certs.Cert.SecretName, configMapDefaultMode))
	}
	if isKeySet(certs.Key) {
		vols = append(vols, secretVolume(prefix+"-tls-key", certs.Key.SecretName, configMapDefaultMode))
	}
	return vols
}

func outputTLSMounts(prefix, basePath string, certs loggingService.Certificates) []corev1.VolumeMount {
	var mounts []corev1.VolumeMount
	if isCASet(certs.CA) {
		mounts = append(mounts, corev1.VolumeMount{Name: prefix + "-tls-ca", MountPath: basePath + "/ca.crt", ReadOnly: true, SubPath: certs.CA.SecretKey})
	}
	if isCertSet(certs.Cert) {
		mounts = append(mounts, corev1.VolumeMount{Name: prefix + "-tls-cert", MountPath: basePath + "/tls.crt", ReadOnly: true, SubPath: certs.Cert.SecretKey})
	}
	if isKeySet(certs.Key) {
		mounts = append(mounts, corev1.VolumeMount{Name: prefix + "-tls-key", MountPath: basePath + "/tls.key", ReadOnly: true, SubPath: certs.Key.SecretKey})
	}
	return mounts
}

// authEnv emits USERNAME+PASSWORD when both are set, else TOKEN if set. Used by all
// three aggregator outputs (Loki, Http, Otel) — same pattern as non-HA fluentbit.
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

func secretVolume(name, secretName string, mode int32) corev1.Volume {
	m := mode
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: &m,
			},
		},
	}
}

func isSecretKeySet(sel *corev1.SecretKeySelector) bool {
	return sel != nil && sel.Name != "" && sel.Key != ""
}
func isCASet(c *loggingService.CA) bool {
	return c != nil && c.SecretName != "" && c.SecretKey != ""
}
func isCertSet(c *loggingService.Cert) bool {
	return c != nil && c.SecretName != "" && c.SecretKey != ""
}
func isKeySet(k *loggingService.Key) bool {
	return k != nil && k.SecretName != "" && k.SecretKey != ""
}
