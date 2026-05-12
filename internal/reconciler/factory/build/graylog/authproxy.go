package graylog

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	corev1 "k8s.io/api/core/v1"
)

// authProxyVolumes returns the pod-level volumes the optional auth-proxy sidecar
// expects: a config-map config volume, the htpasswd secret, and optional CA/Cert/Key
// secret volumes. Returns nil when AuthProxy is unset or Install=false.
func authProxyVolumes(spec *loggingService.Graylog) []corev1.Volume {
	if spec.AuthProxy == nil || !spec.AuthProxy.Install {
		return nil
	}
	ap := spec.AuthProxy
	cmMode := configMapStrictMode
	vols := []corev1.Volume{
		configMapVolume("graylog-auth-proxy-config", "graylog-auth-proxy-config", []corev1.KeyToPath{{Key: "config.yaml", Path: "config.yaml"}}, &cmMode),
	}
	if ap.BindPasswordSecret != nil && ap.BindPasswordSecret.Name != "" {
		vols = append(vols, secretVolume("graylog-auth-proxy-htpasswd", ap.BindPasswordSecret.Name, configMapDefaultMode))
	}
	if ap.CA.SecretName != "" && ap.CA.SecretKey != "" {
		vols = append(vols, secretVolume("graylog-auth-proxy-ca-cert", ap.CA.SecretName, configMapDefaultMode))
	}
	if ap.Cert.SecretName != "" && ap.Cert.SecretKey != "" {
		vols = append(vols, secretVolume("graylog-auth-proxy-client-cert", ap.Cert.SecretName, configMapDefaultMode))
	}
	if ap.Key.SecretName != "" && ap.Key.SecretKey != "" {
		vols = append(vols, secretVolume("graylog-auth-proxy-private-key", ap.Key.SecretName, configMapDefaultMode))
	}
	return vols
}

// buildAuthProxyContainer constructs the optional auth-proxy sidecar. Its container is
// only added when Install=true. The legacy asset places this last in the containers
// list, so callers should append after mongo + graylog.
func buildAuthProxyContainer(cr *loggingService.LoggingService, def config.GraylogDefaults) corev1.Container {
	ap := cr.Spec.Graylog.AuthProxy
	var resources corev1.ResourceRequirements
	if ap.Resources != nil {
		resources = *ap.Resources.DeepCopy()
	}
	runAsNonRoot := true
	runAsUser := int64(1001)
	var sc *corev1.SecurityContext
	if !cr.Spec.OpenshiftDeploy {
		sc = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
	}
	return build.NewContainer(AuthProxyContainer, build.ContainerOpts{
		Image:           ap.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", "python /usr/src/app/graylog_auth_proxy.py --config \"./config.yaml\"\n"},
		VolumeMounts:    authProxyMounts(cr.Spec.Graylog),
		Ports: []corev1.ContainerPort{
			{Name: "proxy", ContainerPort: def.AuthProxyHTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "metrics", ContainerPort: def.AuthProxyMetricsPort, Protocol: corev1.ProtocolTCP},
		},
		Resources:       resources,
		SecurityContext: sc,
	})
}

func authProxyMounts(spec *loggingService.Graylog) []corev1.VolumeMount {
	ap := spec.AuthProxy
	mounts := []corev1.VolumeMount{
		{Name: "graylog-auth-proxy-config", MountPath: "/usr/src/app/config.yaml", SubPath: "config.yaml", ReadOnly: true},
	}
	if ap.BindPasswordSecret != nil && ap.BindPasswordSecret.Key != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "graylog-auth-proxy-htpasswd",
			MountPath: "/usr/src/app/.htpasswd",
			SubPath:   ap.BindPasswordSecret.Key,
			ReadOnly:  true,
		})
	}
	mounts = append(mounts, authProxyTLSMounts(spec)...)
	if ap.CA.SecretName != "" && ap.CA.SecretKey != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: "graylog-auth-proxy-ca-cert", MountPath: "/usr/share/ssl/auth/ca.crt", SubPath: ap.CA.SecretKey, ReadOnly: true})
	}
	if ap.Cert.SecretName != "" && ap.Cert.SecretKey != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: "graylog-auth-proxy-client-cert", MountPath: "/usr/share/ssl/auth/tls.crt", SubPath: ap.Cert.SecretKey, ReadOnly: true})
	}
	if ap.Key.SecretName != "" && ap.Key.SecretKey != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: "graylog-auth-proxy-private-key", MountPath: "/usr/share/ssl/auth/tls.key", SubPath: ap.Key.SecretKey, ReadOnly: true})
	}
	return mounts
}

// authProxyTLSMounts mirrors the legacy asset: the auth-proxy reuses the main TLS
// material under /usr/share/graylog/data/ssl/http/. CACerts under GenerateCerts maps
// to ca.crt (vs the main container's cert-manager-ca.crt path).
func authProxyTLSMounts(spec *loggingService.Graylog) []corev1.VolumeMount {
	if !tlsHTTPEnabled(spec) {
		return nil
	}
	http := spec.TLS.HTTP
	var mounts []corev1.VolumeMount
	if http.GenerateCerts != nil && http.GenerateCerts.Enabled {
		mounts = append(mounts,
			corev1.VolumeMount{Name: "tls-cert-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.crt", ReadOnly: true, SubPath: "tls.crt"},
			corev1.VolumeMount{Name: "tls-key-http", MountPath: "/usr/share/graylog/data/ssl/http/tls.key", ReadOnly: true, SubPath: "tls.key"},
			corev1.VolumeMount{Name: "tls-cacerts-cert-manager", MountPath: "/usr/share/graylog/data/ssl/http/ca.crt", ReadOnly: true, SubPath: "ca.crt"},
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
		mounts = append(mounts, corev1.VolumeMount{Name: "tls-cacerts", MountPath: "/usr/share/graylog/data/ssl/http/ca.crt", ReadOnly: true, SubPath: "ca.crt"})
	}
	return mounts
}
