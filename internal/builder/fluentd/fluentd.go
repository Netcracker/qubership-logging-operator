// Package fluentd builds the FluentD DaemonSet and Service from the LoggingService
// custom resource. The FluentD configuration Secret stays in internal/controllers/fluentd.
package fluentd

import (
	"strconv"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	name            = "logging-fluentd"
	reloadContainer = "configmap-reload"
	technology      = "ruby"
	configVolume    = "config"
	tmpVolume       = "tmp"
	configDir       = "/fluentd/etc"
	// rootID is the user and group FluentD runs as: it must traverse node log
	// directories such as /var/log/pods that are owned by root with mode 0750.
	rootID = int64(0)
	// reloadID is the non-root user and group of the configmap-reload sidecar.
	reloadID = int64(1001)
	// tmpGroupID keeps a stable group on files FluentD writes to the shared /tmp emptyDir.
	tmpGroupID = int64(1000)
	// secretMode grants read and write access to the files of mounted Secrets.
	secretMode = int32(420)
)

// startScript runs FluentD with a private temporary directory. Ruby's Dir.tmpdir
// rejects a world-writable directory without the sticky bit, so FluentD uses a
// subdirectory instead of changing permissions on the shared /tmp emptyDir.
const startScript = `mkdir -p -m 0700 /tmp/fluentd-work
export TMPDIR=/tmp/fluentd-work
exec tini -- /bin/entrypoint.sh /fluentd/entrypoint.sh
`

// selectorLabels link the DaemonSet to its pods.
func selectorLabels(openshift bool) map[string]string {
	provider := "kubernetes"
	if openshift {
		provider = "openshift"
	}
	return map[string]string{"component": name, "provider": provider}
}

// DaemonSet returns the FluentD DaemonSet. The caller checks that cr.Spec.Fluentd is
// set and copies the detected container runtime to cr.Spec.ContainerRuntimeType.
func DaemonSet(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.DaemonSet {
	spec := cr.Spec.Fluentd
	def := cfg.Fluentd
	openshift := cr.Spec.OpenshiftDeploy

	volumes, mounts := volumes(cr, def)
	podSpec := corev1.PodSpec{
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
		ServiceAccountName:            name,
		SecurityContext:               podSecurityContext(openshift),
		Containers:                    []corev1.Container{reloadContainerSpec(spec, def), fluentdContainer(cr, def, mounts)},
		Volumes:                       volumes,
		NodeSelector:                  builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Tolerations:                   builder.FirstSlice(spec.Tolerations, def.Tolerations),
		Affinity:                      spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	labels := selectorLabels(openshift)
	labels["app.kubernetes.io/technology"] = technology
	podLabels := selectorLabels(openshift)
	podLabels["app.kubernetes.io/technology"] = technology

	daemonSet := builder.NewDaemonSet(name, cr.GetNamespace(), builder.DaemonSetOpts{
		Selector:        selectorLabels(openshift),
		MinReadySeconds: 60,
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type:          appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{MaxUnavailable: ptr.To(intstr.FromInt32(1))},
		},
		PodSpec:        podSpec,
		PodLabels:      podLabels,
		PodAnnotations: spec.Annotations,
		Labels:         labels,
		Annotations:    spec.Annotations,
	})
	daemonSet.Spec.Template.Name = name
	util.SetLabelsForWorkload(daemonSet, &daemonSet.Spec.Template.Labels, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		Instance:        util.GetInstanceLabel(name, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return daemonSet
}

// Service returns the FluentD Service. The caller checks that cr.Spec.Fluentd is set.
func Service(cr *loggingService.LoggingService, cfg *defaults.Defaults) *corev1.Service {
	port := cfg.Fluentd.HTTPPort
	service := builder.NewService(name, cr.GetNamespace(), builder.ServiceOpts{
		Labels:   selectorLabels(cr.Spec.OpenshiftDeploy),
		Selector: map[string]string{"name": name},
		Ports: []corev1.ServicePort{{
			Name:       name,
			Port:       port,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt32(port),
		}},
	})
	util.SetLabelsForResource(service, util.LabelInput{
		Name:            name,
		Component:       util.ComponentBackend,
		ComponentLabels: cr.Spec.Fluentd.Labels,
	}, nil)
	return service
}

func reloadContainerSpec(spec *loggingService.Fluentd, def defaults.FluentdDefaults) corev1.Container {
	image := ""
	resources := &def.ConfigmapReloadResources
	if spec.ConfigmapReload != nil {
		image = spec.ConfigmapReload.DockerImage
		resources = builder.FirstNonNil(spec.ConfigmapReload.Resources, resources)
	}
	return builder.NewContainer(reloadContainer, builder.ContainerOpts{
		Image: image,
		Args: []string{
			"--volume-dir=" + configDir,
			"--webhook-url=http://localhost:24444/api/config.reload",
		},
		Ports:     []corev1.ContainerPort{{Name: "metrics", ContainerPort: def.ConfigmapReloadPort}},
		Resources: *resources.DeepCopy(),
		VolumeMounts: []corev1.VolumeMount{
			{Name: configVolume, MountPath: configDir, ReadOnly: true},
			{Name: tmpVolume, MountPath: "/tmp"},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			RunAsNonRoot:             ptr.To(true),
			RunAsUser:                ptr.To(reloadID),
			RunAsGroup:               ptr.To(reloadID),
		},
	})
}

func fluentdContainer(cr *loggingService.LoggingService, def defaults.FluentdDefaults, mounts []corev1.VolumeMount) corev1.Container {
	spec := cr.Spec.Fluentd
	securityContext := &corev1.SecurityContext{
		Privileged:             ptr.To(spec.SecurityContextPrivileged),
		ReadOnlyRootFilesystem: ptr.To(true),
		RunAsNonRoot:           ptr.To(false),
		RunAsUser:              ptr.To(rootID),
		RunAsGroup:             ptr.To(rootID),
	}
	if !spec.SecurityContextPrivileged {
		securityContext.AllowPrivilegeEscalation = ptr.To(false)
		securityContext.Capabilities = &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}
	}
	return builder.NewContainer(name, builder.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", startScript},
		SecurityContext: securityContext,
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: def.HTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "metrics", ContainerPort: def.MetricsPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:  def.LivenessProbe.DeepCopy(),
		ReadinessProbe: def.ReadinessProbe.DeepCopy(),
		Resources:      *builder.FirstNonNil(spec.Resources, &def.Resources).DeepCopy(),
		Env:            env(cr, def),
		VolumeMounts:   mounts,
	})
}

func env(cr *loggingService.LoggingService, def defaults.FluentdDefaults) []corev1.EnvVar {
	spec := cr.Spec.Fluentd
	vars := []corev1.EnvVar{{
		Name:      "K8S_NODE_NAME",
		ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}},
	}}
	if spec.GraylogOutput {
		vars = append(vars,
			corev1.EnvVar{Name: "GRAYLOG_HOST", Value: spec.GraylogHost},
			corev1.EnvVar{Name: "GRAYLOG_PORT", Value: strconv.Itoa(spec.GraylogPort)},
			corev1.EnvVar{Name: "GRAYLOG_PROTOCOL", Value: builder.First(spec.GraylogProtocol, def.GraylogProtocol)},
		)
	}
	listenHost := "0.0.0.0"
	if cr.Spec.Ipv6 {
		listenHost = "::"
	}
	return append(vars,
		corev1.EnvVar{Name: "QUEUE_LIMIT_LENGTH", Value: strconv.Itoa(builder.First(spec.QueueLimitLength, def.QueueLimitLength))},
		// Kubernetes metadata enrichment is always on; the field only exists for compatibility.
		corev1.EnvVar{Name: "WATCH_KUBERNETES_METADATA", Value: "true"},
		corev1.EnvVar{Name: "MA_HOST", Value: listenHost},
		corev1.EnvVar{Name: "RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR", Value: "1.2"},
		corev1.EnvVar{Name: "RUBY_GC_MALLOC_LIMIT", Value: "4194304"},
		corev1.EnvVar{Name: "RUBY_GC_MALLOC_LIMIT_MAX", Value: "16777216"},
		corev1.EnvVar{Name: "RUBY_GC_OLDMALLOC_LIMIT", Value: "16777216"},
		corev1.EnvVar{Name: "RUBY_GC_OLDMALLOC_LIMIT_MAX", Value: "16777216"},
	)
}

func podSecurityContext(openshift bool) *corev1.PodSecurityContext {
	sc := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		FSGroup:        ptr.To(tmpGroupID),
		RunAsNonRoot:   ptr.To(false),
		RunAsUser:      ptr.To(rootID),
		RunAsGroup:     ptr.To(rootID),
	}
	if openshift {
		// The default container_t SELinux type cannot read or write host files labeled
		// var_log_t, such as the /var/log hostPath. spc_t is the standard type for host
		// log collectors on SELinux-enforcing OpenShift hosts.
		sc.SELinuxOptions = &corev1.SELinuxOptions{Type: "spc_t"}
	}
	return sc
}

// volumes returns the pod volumes and the FluentD container mounts in the order the
// removed YAML template used, so that existing pods are not restarted by a reorder.
func volumes(cr *loggingService.LoggingService, def defaults.FluentdDefaults) ([]corev1.Volume, []corev1.VolumeMount) {
	spec := cr.Spec.Fluentd
	var vols []corev1.Volume
	var mounts []corev1.VolumeMount
	hostPath := func(volume, path string, pathType corev1.HostPathType) {
		vols = append(vols, corev1.Volume{
			Name:         volume,
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path, Type: ptr.To(pathType)}},
		})
	}
	secret := func(volume, secretName string) {
		vols = append(vols, corev1.Volume{
			Name: volume,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: ptr.To(secretMode),
			}},
		})
	}
	mount := func(volume, path, subPath string, readOnly bool) {
		mounts = append(mounts, corev1.VolumeMount{Name: volume, MountPath: path, SubPath: subPath, ReadOnly: readOnly})
	}

	if cr.Spec.ContainerRuntimeType == "docker" {
		if cr.Spec.OSKind != "ubuntu" {
			hostPath("sysconfigdocker", "/etc/sysconfig/docker", corev1.HostPathFileOrCreate)
			mount("sysconfigdocker", "/etc/sysconfig/docker", "", true)
		}
		hostPath("dockerdaemon", "/etc/docker/daemon.json", corev1.HostPathFileOrCreate)
		mount("dockerdaemon", "/etc/docker/daemon.json", "", true)
		hostPath("dockerhostname", "/etc/hostname", corev1.HostPathFileOrCreate)
		mount("dockerhostname", "/etc/docker-hostname", "", true)
		hostPath("varlibdockercontainers", "/var/lib/docker/containers", corev1.HostPathUnset)
		mount("varlibdockercontainers", "/var/lib/docker/containers", "", true)
	}
	// FluentD keeps its position files next to the logs it tails, so /var/log is writable.
	hostPath("varlog", "/var/log", corev1.HostPathUnset)
	mount("varlog", "/var/log", "", false)
	secret(configVolume, name)
	mount(configVolume, configDir, "", true)
	vols = append(vols, corev1.Volume{
		Name:         tmpVolume,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(def.TmpVolumeSize)}},
	})
	mount(tmpVolume, "/tmp", "", false)

	// The TLS volumes do not depend on tls.enabled: the template mounted them whenever
	// the certificates were configured.
	tls := spec.TLS
	if tls.GenerateCerts != nil && tls.GenerateCerts.Enabled {
		for _, file := range []struct{ volume, file string }{
			{"tls-ca", "ca.crt"}, {"tls-cert", "tls.crt"}, {"tls-key", "tls.key"},
		} {
			mount(file.volume, "/fluentd/tls/"+file.file, file.file, true)
			secret(file.volume, tls.GenerateCerts.SecretName)
		}
	} else {
		certificateVolumes(tls.Certificates, "tls", "/fluentd/tls", secret, mount)
	}

	if output := spec.Output; output != nil {
		if output.Loki != nil && output.Loki.Enabled && output.Loki.TLS != nil && output.Loki.TLS.Enabled {
			certificateVolumes(output.Loki.TLS.Certificates, "loki-tls", "/fluentd/output/loki/tls", secret, mount)
		}
		if output.Http != nil && output.Http.Enabled && output.Http.TLS != nil && output.Http.TLS.Enabled {
			certificateVolumes(output.Http.TLS.Certificates, "http-tls", "/fluentd/output/http/tls", secret, mount)
		}
	}

	vols = append(vols, spec.AdditionalVolumes...)
	mounts = append(mounts, spec.AdditionalVolumeMounts...)
	return vols, mounts
}

// certificateVolumes adds a Secret volume and a mount for each configured certificate
// file: <prefix>-ca, <prefix>-cert, and <prefix>-key mounted as ca.crt, tls.crt, and
// tls.key under dir.
func certificateVolumes(certs loggingService.Certificates, prefix, dir string,
	secret func(volume, secretName string), mount func(volume, path, subPath string, readOnly bool)) {
	add := func(suffix, file, secretName, secretKey string) {
		if secretName == "" || secretKey == "" {
			return
		}
		volume := prefix + "-" + suffix
		secret(volume, secretName)
		mount(volume, dir+"/"+file, secretKey, true)
	}
	if certs.CA != nil {
		add("ca", "ca.crt", certs.CA.SecretName, certs.CA.SecretKey)
	}
	if certs.Cert != nil {
		add("cert", "tls.crt", certs.Cert.SecretName, certs.Cert.SecretKey)
	}
	if certs.Key != nil {
		add("key", "tls.key", certs.Key.SecretName, certs.Key.SecretKey)
	}
}
