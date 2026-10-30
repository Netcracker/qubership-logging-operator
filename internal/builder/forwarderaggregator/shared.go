// Package forwarderaggregator builds the Kubernetes objects of the FluentBit HA mode
// from the LoggingService custom resource: the forwarder DaemonSet and Service, and
// the aggregator StatefulSet and Service.
package forwarderaggregator

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	forwarderName  = "logging-fluentbit-forwarder"
	aggregatorName = "logging-fluentbit-aggregator"
	reloadName     = "configmap-reload"
	technology     = "c"
	// legacyTechnology is the technology label the removed YAML templates set before
	// util.SetLabelsForWorkload replaced it with technology.
	legacyTechnology = "cpp"

	httpPort         = 2020
	logToMetricsPort = 2021
	forwardPort      = 24224
	reloadPort       = 9533

	configVolume  = "config"
	tmpVolume     = "tmp"
	stateVolume   = "fluentbit-state"
	configDir     = "/fluent-bit/etc"
	secretMode    = int32(420)
	nonRootID     = int64(1001)
	tlsCAFile     = "ca.crt"
	tlsCertFile   = "tls.crt"
	tlsKeyFile    = "tls.key"
	tlsVolumeCA   = "tls-ca"
	tlsVolumeCert = "tls-cert"
	tlsVolumeKey  = "tls-key"
)

var tmpVolumeSize = resource.MustParse("100Mi")

// provider returns the value of the provider label.
func provider(openshift bool) string {
	if openshift {
		return "openshift"
	}
	return "kubernetes"
}

// workloadLabels returns the labels the workload and its pod template start with
// before util.SetLabelsForWorkload adds the common ones.
func workloadLabels(component string, openshift bool) map[string]string {
	return map[string]string{
		"app.kubernetes.io/technology":  legacyTechnology,
		"component":                     component,
		"provider":                      provider(openshift),
		"kubernetes.io/cluster-service": "true",
	}
}

// reloadContainer returns the configmap-reload sidecar that asks FluentBit to reload
// its configuration when the mounted files change.
func reloadContainer(reload *loggingService.ConfigmapReload, fallback corev1.ResourceRequirements) corev1.Container {
	image := ""
	resources := &fallback
	if reload != nil {
		image = reload.DockerImage
		resources = builder.FirstNonNil(reload.Resources, &fallback)
	}
	return builder.NewContainer(reloadName, builder.ContainerOpts{
		Image: image,
		Args: []string{
			"--volume-dir=" + configDir,
			"--webhook-url=http://localhost:2020/api/v2/reload",
			"--webhook-retries=5",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: configVolume, MountPath: configDir, ReadOnly: true},
			{Name: tmpVolume, MountPath: "/tmp"},
		},
		Resources:       *resources.DeepCopy(),
		Ports:           []corev1.ContainerPort{{Name: "reload-metrics", ContainerPort: reloadPort}},
		SecurityContext: nonRootSecurityContext(),
	})
}

// nonRootSecurityContext is the hardened context of the containers that do not read
// node logs.
func nonRootSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To(nonRootID),
		RunAsGroup:               ptr.To(nonRootID),
	}
}

func httpProbe(path string, failureThreshold int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Path:   path,
			Port:   intstr.FromInt32(httpPort),
			Scheme: corev1.URISchemeHTTP,
		}},
		InitialDelaySeconds: 5,
		TimeoutSeconds:      10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    failureThreshold,
	}
}

// setMainContainerDefaults sets the fields the removed YAML templates set explicitly
// on the FluentBit containers.
func setMainContainerDefaults(container *corev1.Container) {
	container.TerminationMessagePath = corev1.TerminationMessagePathDefault
	container.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	container.ImagePullPolicy = corev1.PullIfNotPresent
}

func emptyDirVolume(name string, size resource.Quantity) corev1.Volume {
	return corev1.Volume{
		Name:         name,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &size}},
	}
}

func secretVolume(name, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
			SecretName:  secretName,
			DefaultMode: ptr.To(secretMode),
		}},
	}
}

func hostPathVolume(name, path string, pathType corev1.HostPathType) corev1.Volume {
	return corev1.Volume{
		Name:         name,
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path, Type: ptr.To(pathType)}},
	}
}

// secretFile is one file of a certificate set: the Secret holding it and the key in
// that Secret.
type secretFile struct {
	secretName, secretKey string
}

func (f secretFile) set() bool {
	return f.secretName != "" && f.secretKey != ""
}

// certificateFiles returns the CA, certificate, and key files of certs. A file that
// is not configured has an empty Secret name.
func certificateFiles(certs loggingService.Certificates) (ca, cert, key secretFile) {
	if certs.CA != nil {
		ca = secretFile{certs.CA.SecretName, certs.CA.SecretKey}
	}
	if certs.Cert != nil {
		cert = secretFile{certs.Cert.SecretName, certs.Cert.SecretKey}
	}
	if certs.Key != nil {
		key = secretFile{certs.Key.SecretName, certs.Key.SecretKey}
	}
	return ca, cert, key
}

// certificateMounts returns the volumes and mounts of the configured CA, certificate,
// and key files. The volumes are named <prefix>-ca, <prefix>-cert, and <prefix>-key,
// and the files are mounted as ca.crt, tls.crt, and tls.key in dir.
func certificateMounts(prefix, dir string, ca, cert, key secretFile) ([]corev1.Volume, []corev1.VolumeMount) {
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	for _, file := range []struct {
		suffix, path string
		secretFile
	}{
		{"-ca", tlsCAFile, ca},
		{"-cert", tlsCertFile, cert},
		{"-key", tlsKeyFile, key},
	} {
		if !file.set() {
			continue
		}
		name := prefix + file.suffix
		volumes = append(volumes, secretVolume(name, file.secretName))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      name,
			MountPath: dir + "/" + file.path,
			ReadOnly:  true,
			SubPath:   file.secretKey,
		})
	}
	return volumes, mounts
}

// collectorTLSMounts returns the volumes and mounts of the TLS files FluentBit uses for
// the Graylog output. Generated certificates take all three files from one Secret.
// Like the removed templates, it does not check tls.enabled.
func collectorTLSMounts(tls loggingService.FluentbitTLS) ([]corev1.Volume, []corev1.VolumeMount) {
	ca, cert, key := certificateFiles(tls.Certificates)
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		name := tls.GenerateCerts.SecretName
		ca = secretFile{name, tlsCAFile}
		cert = secretFile{name, tlsCertFile}
		key = secretFile{name, tlsKeyFile}
		// The template mounted generated files even when the Secret name is empty.
		return generatedTLSMounts(ca, cert, key)
	}
	return certificateMounts("tls", "/fluent-bit/tls", ca, cert, key)
}

func generatedTLSMounts(files ...secretFile) ([]corev1.Volume, []corev1.VolumeMount) {
	names := []string{tlsVolumeCA, tlsVolumeCert, tlsVolumeKey}
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	for i, file := range files {
		volumes = append(volumes, secretVolume(names[i], file.secretName))
		mounts = append(mounts, corev1.VolumeMount{
			Name:      names[i],
			MountPath: "/fluent-bit/tls/" + file.secretKey,
			ReadOnly:  true,
			SubPath:   file.secretKey,
		})
	}
	return volumes, mounts
}
