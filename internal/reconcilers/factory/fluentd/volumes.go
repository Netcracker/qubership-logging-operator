package fluentd

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	configMapDefaultMode   = int32(420) // 0644 in decimal
	authTokenDefaultMode   = int32(220) // preserved from the legacy asset
	containerRuntimeDocker = "docker"
	osKindUbuntu           = "ubuntu"
)

func buildVolumes(cr *loggingService.LoggingService) []corev1.Volume {
	spec := cr.Spec.Fluentd
	cmMode := configMapDefaultMode
	tokenMode := authTokenDefaultMode

	var vols []corev1.Volume
	vols = append(vols, dockerHostPathVolumes(cr.Spec.ContainerRuntimeType, cr.Spec.OSKind)...)
	vols = append(vols,
		corev1.Volume{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/var/log", Type: hostPathType("")},
			},
		},
		corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: ComponentName},
					DefaultMode:          &cmMode,
				},
			},
		},
	)
	vols = append(vols, tlsVolumes(&spec.TLS)...)
	if spec.Output != nil {
		if spec.Output.Loki != nil && spec.Output.Loki.Enabled {
			if spec.Output.Loki.Auth != nil && isSecretKeySet(spec.Output.Loki.Auth.Token) {
				vols = append(vols, secretVolume("loki-auth-token", spec.Output.Loki.Auth.Token.Name, tokenMode))
			}
			if spec.Output.Loki.TLS != nil && spec.Output.Loki.TLS.Enabled {
				vols = append(vols, outputTLSVolumes("loki", spec.Output.Loki.TLS.Certificates)...)
			}
		}
		if spec.Output.Http != nil && spec.Output.Http.Enabled && spec.Output.Http.TLS != nil && spec.Output.Http.TLS.Enabled {
			vols = append(vols, outputTLSVolumes("http", spec.Output.Http.TLS.Certificates)...)
		}
	}
	if len(spec.AdditionalVolumes) > 0 {
		vols = append(vols, spec.AdditionalVolumes...)
	}
	return vols
}

func buildMainVolumeMounts(cr *loggingService.LoggingService) []corev1.VolumeMount {
	spec := cr.Spec.Fluentd
	var mounts []corev1.VolumeMount
	mounts = append(mounts, dockerVolumeMounts(cr.Spec.ContainerRuntimeType, cr.Spec.OSKind)...)
	mounts = append(mounts,
		corev1.VolumeMount{Name: "varlog", MountPath: "/var/log"},
		corev1.VolumeMount{Name: "config", MountPath: "/fluentd/etc", ReadOnly: true},
	)
	mounts = append(mounts, tlsVolumeMounts(&spec.TLS)...)
	if spec.Output != nil {
		if spec.Output.Loki != nil && spec.Output.Loki.Enabled {
			if spec.Output.Loki.Auth != nil && isSecretKeySet(spec.Output.Loki.Auth.Token) {
				mounts = append(mounts, corev1.VolumeMount{
					Name:      "loki-auth-token",
					MountPath: "/fluentd/output/loki/auth/token",
					ReadOnly:  true,
					SubPath:   spec.Output.Loki.Auth.Token.Key,
				})
			}
			if spec.Output.Loki.TLS != nil && spec.Output.Loki.TLS.Enabled {
				mounts = append(mounts, outputTLSMounts("loki", "/fluentd/output/loki/tls", spec.Output.Loki.TLS.Certificates)...)
			}
		}
		if spec.Output.Http != nil && spec.Output.Http.Enabled && spec.Output.Http.TLS != nil && spec.Output.Http.TLS.Enabled {
			mounts = append(mounts, outputTLSMounts("http", "/fluentd/output/http/tls", spec.Output.Http.TLS.Certificates)...)
		}
	}
	return mounts
}

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

func tlsVolumes(tls *loggingService.FluentdTLS) []corev1.Volume {
	mode := configMapDefaultMode
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		name := tls.GenerateCerts.SecretName
		return []corev1.Volume{
			secretVolume("tls-ca", name, mode),
			secretVolume("tls-cert", name, mode),
			secretVolume("tls-key", name, mode),
		}
	}
	var vols []corev1.Volume
	if isCASet(tls.CA) {
		vols = append(vols, secretVolume("tls-ca", tls.CA.SecretName, mode))
	}
	if isCertSet(tls.Cert) {
		vols = append(vols, secretVolume("tls-cert", tls.Cert.SecretName, mode))
	}
	if isKeySet(tls.Key) {
		vols = append(vols, secretVolume("tls-key", tls.Key.SecretName, mode))
	}
	return vols
}

func tlsVolumeMounts(tls *loggingService.FluentdTLS) []corev1.VolumeMount {
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		return []corev1.VolumeMount{
			{Name: "tls-ca", MountPath: "/fluentd/tls/ca.crt", ReadOnly: true, SubPath: "ca.crt"},
			{Name: "tls-cert", MountPath: "/fluentd/tls/tls.crt", ReadOnly: true, SubPath: "tls.crt"},
			{Name: "tls-key", MountPath: "/fluentd/tls/tls.key", ReadOnly: true, SubPath: "tls.key"},
		}
	}
	var mounts []corev1.VolumeMount
	if isCASet(tls.CA) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-ca", MountPath: "/fluentd/tls/ca.crt", ReadOnly: true, SubPath: tls.CA.SecretKey})
	}
	if isCertSet(tls.Cert) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-cert", MountPath: "/fluentd/tls/tls.crt", ReadOnly: true, SubPath: tls.Cert.SecretKey})
	}
	if isKeySet(tls.Key) {
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-key", MountPath: "/fluentd/tls/tls.key", ReadOnly: true, SubPath: tls.Key.SecretKey})
	}
	return mounts
}

func outputTLSVolumes(prefix string, certs loggingService.Certificates) []corev1.Volume {
	mode := configMapDefaultMode
	var vols []corev1.Volume
	if isCASet(certs.CA) {
		vols = append(vols, secretVolume(prefix+"-tls-ca", certs.CA.SecretName, mode))
	}
	if isCertSet(certs.Cert) {
		vols = append(vols, secretVolume(prefix+"-tls-cert", certs.Cert.SecretName, mode))
	}
	if isKeySet(certs.Key) {
		vols = append(vols, secretVolume(prefix+"-tls-key", certs.Key.SecretName, mode))
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

func hostPathType(t corev1.HostPathType) *corev1.HostPathType { return &t }

func isCASet(c *loggingService.CA) bool {
	return c != nil && c.SecretName != "" && c.SecretKey != ""
}
func isCertSet(c *loggingService.Cert) bool {
	return c != nil && c.SecretName != "" && c.SecretKey != ""
}
func isKeySet(k *loggingService.Key) bool {
	return k != nil && k.SecretName != "" && k.SecretKey != ""
}
