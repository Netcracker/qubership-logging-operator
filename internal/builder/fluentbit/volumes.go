package fluentbit

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	corev1 "k8s.io/api/core/v1"
)

// secretFile is a file mounted from a Secret key into the collector container.
type secretFile struct {
	volume     string
	secretName string
	secretKey  string
	mountPath  string
}

func volumeMounts(cr *loggingService.LoggingService) []corev1.VolumeMount {
	spec := cr.Spec.Fluentbit
	var mounts []corev1.VolumeMount
	if cr.Spec.ContainerRuntimeType == "docker" {
		if cr.Spec.OSKind != "ubuntu" {
			mounts = append(mounts, corev1.VolumeMount{Name: "sysconfigdocker", MountPath: "/etc/sysconfig/docker", ReadOnly: true})
		}
		mounts = append(mounts,
			corev1.VolumeMount{Name: "dockerdaemon", MountPath: "/etc/docker/daemon.json", ReadOnly: true},
			corev1.VolumeMount{Name: "dockerhostname", MountPath: "/etc/docker-hostname", ReadOnly: true},
			corev1.VolumeMount{Name: "varlibdockercontainers", MountPath: "/var/lib/docker/containers", ReadOnly: true},
		)
	}
	mounts = append(mounts,
		corev1.VolumeMount{Name: "varlog", MountPath: "/var/log", ReadOnly: true},
		corev1.VolumeMount{Name: stateVolume, MountPath: "/fluent-bit/state"},
	)
	if spec.UsesFilesystemBuffer() {
		mounts = append(mounts, corev1.VolumeMount{Name: storageVolume, MountPath: "/fluent-bit/storage"})
	}
	mounts = append(mounts,
		corev1.VolumeMount{Name: configVolume, MountPath: "/fluent-bit/etc", ReadOnly: true},
		corev1.VolumeMount{Name: tmpVolume, MountPath: "/tmp"},
	)
	for _, file := range secretFiles(spec) {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      file.volume,
			MountPath: file.mountPath,
			ReadOnly:  true,
			SubPath:   file.secretKey,
		})
	}
	return mounts
}

func volumes(cr *loggingService.LoggingService, def defaults.FluentbitDefaults) []corev1.Volume {
	spec := cr.Spec.Fluentbit
	var volumes []corev1.Volume
	if cr.Spec.ContainerRuntimeType == "docker" {
		if cr.Spec.OSKind != "ubuntu" {
			volumes = append(volumes, hostPathVolume("sysconfigdocker", "/etc/sysconfig/docker", corev1.HostPathFileOrCreate))
		}
		volumes = append(volumes,
			hostPathVolume("dockerdaemon", "/etc/docker/daemon.json", corev1.HostPathFileOrCreate),
			hostPathVolume("dockerhostname", "/etc/hostname", corev1.HostPathFileOrCreate),
			hostPathVolume("varlibdockercontainers", "/var/lib/docker/containers", corev1.HostPathUnset),
		)
	}
	volumes = append(volumes, hostPathVolume("varlog", "/var/log", corev1.HostPathUnset))

	if spec.UsesPersistentOffsets() {
		volumes = append(volumes, hostPathVolume(stateVolume, "/var/lib/fluent-bit/state", corev1.HostPathDirectoryOrCreate))
	} else {
		limit := stateSizeLimit(spec)
		volumes = append(volumes, corev1.Volume{
			Name: stateVolume,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: &limit,
			}},
		})
	}
	if spec.UsesFilesystemBuffer() {
		volumes = append(volumes, hostPathVolume(storageVolume, "/var/lib/fluent-bit/storage", corev1.HostPathDirectoryOrCreate))
	}

	tmpSize := def.TmpVolumeSize.DeepCopy()
	volumes = append(volumes,
		secretVolume(configVolume, name),
		corev1.Volume{
			Name:         tmpVolume,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &tmpSize}},
		},
	)
	for _, file := range secretFiles(spec) {
		volumes = append(volumes, secretVolume(file.volume, file.secretName))
	}
	return volumes
}

// secretFiles lists the TLS files of the Graylog output and of the Loki, HTTP, and
// OpenTelemetry outputs, in the order the collector mounts them.
func secretFiles(spec *loggingService.Fluentbit) []secretFile {
	var files []secretFile
	tls := spec.TLS
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		secretName := tls.GenerateCerts.SecretName
		files = append(files,
			secretFile{"tls-ca", secretName, "ca.crt", "/fluent-bit/tls/ca.crt"},
			secretFile{"tls-cert", secretName, "tls.crt", "/fluent-bit/tls/tls.crt"},
			secretFile{"tls-key", secretName, "tls.key", "/fluent-bit/tls/tls.key"},
		)
	} else {
		files = append(files, certificateFiles("tls", "/fluent-bit/tls", tls.Certificates)...)
	}

	output := spec.Output
	if output == nil {
		return files
	}
	if output.Loki != nil && output.Loki.Enabled && output.Loki.TLS != nil && output.Loki.TLS.Enabled {
		files = append(files, certificateFiles("loki-tls", "/fluent-bit/output/loki/tls", output.Loki.TLS.Certificates)...)
	}
	if output.Http != nil && output.Http.Enabled && output.Http.TLS != nil && output.Http.TLS.Enabled {
		files = append(files, certificateFiles("http-tls", "/fluent-bit/output/http/tls", output.Http.TLS.Certificates)...)
	}
	if output.Otel != nil && output.Otel.Enabled && output.Otel.TLS != nil && output.Otel.TLS.Enabled {
		files = append(files, certificateFiles("otel-tls", "/fluent-bit/output/otel/tls", output.Otel.TLS.Certificates)...)
	}
	return files
}

// certificateFiles returns the CA, certificate, and key files that have both a Secret
// name and a key.
func certificateFiles(prefix, dir string, certs loggingService.Certificates) []secretFile {
	var files []secretFile
	if certs.CA != nil && certs.CA.SecretName != "" && certs.CA.SecretKey != "" {
		files = append(files, secretFile{prefix + "-ca", certs.CA.SecretName, certs.CA.SecretKey, dir + "/ca.crt"})
	}
	if certs.Cert != nil && certs.Cert.SecretName != "" && certs.Cert.SecretKey != "" {
		files = append(files, secretFile{prefix + "-cert", certs.Cert.SecretName, certs.Cert.SecretKey, dir + "/tls.crt"})
	}
	if certs.Key != nil && certs.Key.SecretName != "" && certs.Key.SecretKey != "" {
		files = append(files, secretFile{prefix + "-key", certs.Key.SecretName, certs.Key.SecretKey, dir + "/tls.key"})
	}
	return files
}

func hostPathVolume(volumeName, path string, pathType corev1.HostPathType) corev1.Volume {
	return corev1.Volume{
		Name:         volumeName,
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path, Type: &pathType}},
	}
}

func secretVolume(volumeName, secretName string) corev1.Volume {
	mode := secretFileMode
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
			SecretName:  secretName,
			DefaultMode: &mode,
		}},
	}
}
