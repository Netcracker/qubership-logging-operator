package graylog

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// tlsHTTPVolumes returns the pod-level Secret volumes for the HTTP TLS material. The
// legacy asset emits three logical volumes (cert, key, cert-manager CA) when
// GenerateCerts is enabled (all pointing at the same secret); otherwise individual
// Cert and Key volumes are emitted per their SecretName, plus an optional CACerts
// bundle volume.
func tlsHTTPVolumes(spec *loggingService.Graylog) []corev1.Volume {
	if !tlsHTTPEnabled(spec) {
		return nil
	}
	http := spec.TLS.HTTP
	var vols []corev1.Volume
	if http.GenerateCerts != nil && http.GenerateCerts.Enabled {
		name := http.GenerateCerts.SecretName
		vols = append(vols,
			secretVolume("tls-cert-http", name, configMapDefaultMode),
			secretVolume("tls-key-http", name, configMapDefaultMode),
			secretVolume("tls-cacerts-cert-manager", name, configMapDefaultMode),
		)
	} else {
		if http.Cert != nil && http.Cert.SecretName != "" && http.Cert.SecretKey != "" {
			vols = append(vols, secretVolume("tls-cert-http", http.Cert.SecretName, configMapDefaultMode))
		}
		if http.Key != nil && http.Key.SecretName != "" && http.Key.SecretKey != "" {
			vols = append(vols, secretVolume("tls-key-http", http.Key.SecretName, configMapDefaultMode))
		}
	}
	if http.CACerts != "" {
		vols = append(vols, secretVolume("tls-cacerts", http.CACerts, configMapDefaultMode))
	}
	return vols
}

// tlsInputVolumes returns Secret volumes for the Input TLS material. Only Cert+Key are
// modelled; the GenerateCerts branch shares the secret across both volumes.
func tlsInputVolumes(spec *loggingService.Graylog) []corev1.Volume {
	if spec.TLS == nil || spec.TLS.Input == nil {
		return nil
	}
	input := spec.TLS.Input
	if input.GenerateCerts != nil && input.GenerateCerts.Enabled {
		name := input.GenerateCerts.SecretName
		return []corev1.Volume{
			secretVolume("tls-cert-input", name, configMapDefaultMode),
			secretVolume("tls-key-input", name, configMapDefaultMode),
		}
	}
	var vols []corev1.Volume
	if input.Cert != nil && input.Cert.SecretName != "" && input.Cert.SecretKey != "" {
		vols = append(vols, secretVolume("tls-cert-input", input.Cert.SecretName, configMapDefaultMode))
	}
	if input.Key != nil && input.Key.SecretName != "" && input.Key.SecretKey != "" {
		vols = append(vols, secretVolume("tls-key-input", input.Key.SecretName, configMapDefaultMode))
	}
	return vols
}

// graylogTLSMounts returns the TLS volume mounts on the main graylog container.
// HTTP first (GenerateCerts vs per-secret SubPath, plus CACerts directory mount),
// then Input.
func graylogTLSMounts(spec *loggingService.Graylog) []corev1.VolumeMount {
	var mounts []corev1.VolumeMount
	if tlsHTTPEnabled(spec) {
		http := spec.TLS.HTTP
		if http.GenerateCerts != nil && http.GenerateCerts.Enabled {
			mounts = append(mounts,
				corev1.VolumeMount{Name: "tls-cert-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.crt", ReadOnly: true, SubPath: "tls.crt"},
				corev1.VolumeMount{Name: "tls-key-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.key", ReadOnly: true, SubPath: "tls.key"},
				corev1.VolumeMount{Name: "tls-cacerts-cert-manager", MountPath: "/usr/share/graylog/data/ssl/cacerts/cert-manager-ca.crt", ReadOnly: true, SubPath: "ca.crt"},
			)
		} else {
			if http.Cert != nil && http.Cert.SecretName != "" && http.Cert.SecretKey != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: "tls-cert-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.crt", ReadOnly: true, SubPath: http.Cert.SecretKey})
			}
			if http.Key != nil && http.Key.SecretName != "" && http.Key.SecretKey != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: "tls-key-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.key", ReadOnly: true, SubPath: http.Key.SecretKey})
			}
		}
		if http.CACerts != "" {
			mounts = append(mounts, corev1.VolumeMount{Name: "tls-cacerts", MountPath: "/usr/share/graylog/data/ssl/cacerts", ReadOnly: true})
		}
	}
	if spec.TLS != nil && spec.TLS.Input != nil {
		input := spec.TLS.Input
		if input.GenerateCerts != nil && input.GenerateCerts.Enabled {
			mounts = append(mounts,
				corev1.VolumeMount{Name: "tls-cert-input", MountPath: "/usr/share/graylog/data/ssl/input/tls.crt", ReadOnly: true, SubPath: "tls.crt"},
				corev1.VolumeMount{Name: "tls-key-input", MountPath: "/usr/share/graylog/data/ssl/input/tls.key", ReadOnly: true, SubPath: "tls.key"},
			)
		} else {
			if input.Cert != nil && input.Cert.SecretName != "" && input.Cert.SecretKey != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: "tls-cert-input", MountPath: "/usr/share/graylog/data/ssl/input/tls.crt", ReadOnly: true, SubPath: input.Cert.SecretKey})
			}
			if input.Key != nil && input.Key.SecretName != "" && input.Key.SecretKey != "" {
				mounts = append(mounts, corev1.VolumeMount{Name: "tls-key-input", MountPath: "/usr/share/graylog/data/ssl/input/tls.key", ReadOnly: true, SubPath: input.Key.SecretKey})
			}
		}
	}
	return mounts
}
