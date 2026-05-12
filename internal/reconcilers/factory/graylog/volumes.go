package graylog

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	configMapDefaultMode = int32(420) // 0644
	configMapStrictMode  = int32(0444)
	nodeIDMode           = int32(0666)
)

// buildPodVolumes assembles the StatefulSet pod-level volumes in the exact order the
// legacy asset wrote them: data PVC, plugins emptyDir, mongodb PVC, three configmap
// projections, AuthProxy volumes (when installed), and TLS HTTP+Input volumes.
func buildPodVolumes(cr *loggingService.LoggingService) []corev1.Volume {
	spec := cr.Spec.Graylog
	cmMode := configMapStrictMode
	nodeMode := nodeIDMode

	vols := []corev1.Volume{
		{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: GraylogClaim}}},
		{Name: "plugins", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "mongodb", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: MongoClaim}}},
		configMapVolume("logsconf", ServiceName, []corev1.KeyToPath{{Key: "log4j2.xml", Path: "log4j2.xml"}}, &cmMode),
		configMapVolume("graylogconf", ServiceName, []corev1.KeyToPath{{Key: "graylog.conf", Path: "graylog.conf"}}, &cmMode),
		configMapVolume("nodeid", ServiceName, []corev1.KeyToPath{{Key: "node-id", Path: "node-id"}}, &nodeMode),
	}
	vols = append(vols, authProxyVolumes(spec)...)
	vols = append(vols, tlsHTTPVolumes(spec)...)
	vols = append(vols, tlsInputVolumes(spec)...)
	return vols
}

func configMapVolume(name, cmName string, items []corev1.KeyToPath, mode *int32) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Items:                items,
				DefaultMode:          mode,
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
