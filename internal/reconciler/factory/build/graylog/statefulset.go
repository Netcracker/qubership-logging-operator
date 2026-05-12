package graylog

import (
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func buildStatefulSet(cr *loggingService.LoggingService, def config.GraylogDefaults) *appsv1.StatefulSet {
	spec := cr.Spec.Graylog

	initContainers := []corev1.Container{buildSetupInit(spec, def)}
	if spec.InitContainerDockerImage != "" {
		initContainers = append(initContainers, buildDownloadPluginsInit(spec, def))
	}

	containers := []corev1.Container{
		buildMongoContainer(cr, def),
		buildGraylogContainer(cr, def),
	}
	if spec.AuthProxy != nil && spec.AuthProxy.Install {
		containers = append(containers, buildAuthProxyContainer(cr, def))
	}

	pod := corev1.PodSpec{
		ServiceAccountName: ServiceAccountName,
		SecurityContext:    def.PodSecurityContext.DeepCopy(),
		Volumes:            buildPodVolumes(cr),
		InitContainers:     initContainers,
		Containers:         containers,
		NodeSelector:       nodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		Affinity:           spec.Affinity,
		PriorityClassName:  spec.PriorityClassName,
	}

	replicas := def.Replicas
	if spec.Replicas != nil {
		replicas = int32(*spec.Replicas)
	}

	ss := build.NewStatefulSet(StatefulSetName, cr.GetNamespace(), "graylog", build.StatefulSetOpts{
		Replicas: &replicas,
		Selector: map[string]string{"name": StatefulSetName},
		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
		},
		PodSpec:        pod,
		PodLabels:      map[string]string{"app.kubernetes.io/technology": Technology},
		PodAnnotations: spec.Annotations,
		ExtraLabels:    map[string]string{"app.kubernetes.io/technology": Technology},
		Annotations:    spec.Annotations,
	})

	util.SetLabelsForWorkload(ss, &ss.Spec.Template.Labels, util.LabelInput{
		Name:            StatefulSetName,
		Component:       "graylog",
		Instance:        util.GetInstanceLabel(StatefulSetName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      Technology,
		ComponentLabels: spec.Labels,
	})
	return ss
}

// graylogVersionForImage returns "5" for Graylog 5.x images and "4" otherwise; passed
// to the download-plugins init container so it picks the right plugin tarball.
func graylogVersionForImage(image string) string {
	if IsV5(image) {
		return "5"
	}
	return "4"
}

func nodeSelector(k, v string) map[string]string {
	if k == "" || v == "" {
		return nil
	}
	return map[string]string{k: v}
}

// buildSetupInit is the init container that prepares /usr/share/graylog/data: chowns
// to 1100:1100, creates archives + config directories, and (when TLS+HTTP is
// configured) creates an ssl directory for the graylog container to write the
// keystore into.
func buildSetupInit(spec *loggingService.Graylog, def config.GraylogDefaults) corev1.Container {
	resources := def.InitResources.DeepCopy()
	if spec.InitResources != nil {
		resources = spec.InitResources.DeepCopy()
	}
	runAsNonRoot := false
	return build.NewContainer(SetupInit, build.ContainerOpts{
		Image:           spec.InitSetupImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/sh",
			"-c",
			setupInitScript(spec),
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: "/usr/share/graylog/data"},
		},
		Resources:       *resources,
		SecurityContext: &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot},
	})
}

func setupInitScript(spec *loggingService.Graylog) string {
	var b strings.Builder
	b.WriteString(`chmod -R 0777 /usr/share/graylog/data/
chown -R 1100:1100 /usr/share/graylog/data
mkdir /usr/share/graylog/data/archives
chown -R 1100:1100 /usr/share/graylog/data/archives
mkdir /usr/share/graylog/data/config
touch /usr/share/graylog/data/config/directories.json
chmod 0666 /usr/share/graylog/data/config/directories.json
chown 1100:1100 /usr/share/graylog/data/config/directories.json
echo "{}" >> /usr/share/graylog/data/config/directories.json
`)
	if tlsHTTPEnabled(spec) {
		b.WriteString(`mkdir /usr/share/graylog/data/ssl
chmod -R 0777 /usr/share/graylog/data/ssl
chown -R 1100:1100 /usr/share/graylog/data/ssl
`)
	}
	return b.String()
}

// buildDownloadPluginsInit is added only when InitContainerDockerImage is set on the
// CR. The CUSTOM_PLUGINS env is sourced from CustomPluginsPaths (empty when unset);
// the GRAYLOG_VERSION env (4 or 5) is appended by the controller after build to keep
// the version-detection regex co-located with checkGraylog5.
func buildDownloadPluginsInit(spec *loggingService.Graylog, def config.GraylogDefaults) corev1.Container {
	resources := def.InitResources.DeepCopy()
	if spec.InitResources != nil {
		resources = spec.InitResources.DeepCopy()
	}
	runAsNonRoot := true
	runAsUser := int64(1001)
	return build.NewContainer(DownloadPluginsInit, build.ContainerOpts{
		Image:           spec.InitContainerDockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "plugins", MountPath: "/opt/plugins"},
		},
		Env: []corev1.EnvVar{
			{Name: "CUSTOM_PLUGINS", Value: spec.CustomPluginsPaths},
			{Name: "GRAYLOG_VERSION", Value: graylogVersionForImage(spec.DockerImage)},
		},
		Resources: *resources,
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
		},
	})
}

func buildMongoContainer(cr *loggingService.LoggingService, def config.GraylogDefaults) corev1.Container {
	spec := cr.Spec.Graylog
	resources := def.MongoResources.DeepCopy()
	if spec.MongoResources != nil {
		resources = spec.MongoResources.DeepCopy()
	}
	c := build.NewContainer(MongoContainer, build.ContainerOpts{
		Image:           spec.MongoDBImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", "mongod --wiredTigerEngineConfigString=\"cache_size=512M\"\n"},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "mongodb", MountPath: "/data/db"},
		},
		Ports: []corev1.ContainerPort{
			{Name: "mongo", ContainerPort: def.MongoPort, Protocol: corev1.ProtocolTCP},
		},
		Resources:       *resources,
		SecurityContext: mongoSecurityContext(cr.Spec.OpenshiftDeploy),
	})
	return c
}

// mongoSecurityContext returns runAsNonRoot=true, runAsUser=1001 on non-OpenShift
// clusters; on OpenShift the SCC handles it and the asset leaves it empty.
func mongoSecurityContext(openshift bool) *corev1.SecurityContext {
	if openshift {
		return nil
	}
	runAsNonRoot := true
	runAsUser := int64(1001)
	return &corev1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &runAsUser,
	}
}

func buildGraylogContainer(cr *loggingService.LoggingService, def config.GraylogDefaults) corev1.Container {
	spec := cr.Spec.Graylog
	resources := def.GraylogResources.DeepCopy()
	if spec.GraylogResources != nil {
		resources = spec.GraylogResources.DeepCopy()
	}
	scheme := tlsHTTPListenScheme(spec)
	liveness := def.LivenessProbe.DeepCopy()
	liveness.ProbeHandler.HTTPGet.Scheme = scheme
	readiness := def.ReadinessProbe.DeepCopy()
	readiness.ProbeHandler.HTTPGet.Scheme = scheme

	httpPort := def.HTTPPort
	udpPort := def.UDPPort
	metricsPort := def.MetricsPort
	inputPort := int32(spec.InputPort)

	runAsNonRoot := true
	runAsUser := int64(1100)
	var sc *corev1.SecurityContext
	if !cr.Spec.OpenshiftDeploy {
		sc = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
	}

	return build.NewContainer(GraylogContainer, build.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", graylogContainerScript(spec)},
		Env:             graylogContainerEnv(spec, def),
		Ports: []corev1.ContainerPort{
			{Name: "graylog", ContainerPort: httpPort, Protocol: corev1.ProtocolTCP},
			{Name: "graylog-udp", ContainerPort: udpPort, Protocol: corev1.ProtocolUDP},
			{Name: graylogInputPortName(inputPort), ContainerPort: inputPort, Protocol: corev1.ProtocolTCP},
			{Name: "graylog-metrics", ContainerPort: metricsPort, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe:   liveness,
		ReadinessProbe:  readiness,
		Resources:       *resources,
		VolumeMounts:    graylogContainerMounts(spec),
		SecurityContext: sc,
	})
}

// graylogContainerScript builds the multi-line shell command Graylog runs at startup.
// When TLS+HTTP is configured, it imports the TLS cert into a JKS truststore before
// invoking the entrypoint; when GenerateCerts is enabled or CACerts is set, additional
// keytool invocations import those CAs too. Mirrors the legacy asset 1:1.
func graylogContainerScript(spec *loggingService.Graylog) string {
	var b strings.Builder
	b.WriteString(`if [ -f /tmp/kafka-logs/.lock ] ; then
    rm /tmp/kafka-logs/.lock
fi
`)
	if tlsHTTPEnabled(spec) {
		b.WriteString(`cp -a "/opt/java/openjdk/lib/security/cacerts" "/usr/share/graylog/data/ssl/cacerts.jks"
keytool -importcert -keystore /usr/share/graylog/data/ssl/cacerts.jks -storepass changeit -alias graylog-tls-ca -file /usr/share/graylog/data/ssl/http/tls.crt -noprompt
`)
		if spec.TLS.HTTP.GenerateCerts != nil && spec.TLS.HTTP.GenerateCerts.Enabled {
			b.WriteString("keytool -importcert -keystore /usr/share/graylog/data/ssl/cacerts.jks -storepass changeit -alias graylog-tls-ca-cert-manager -file /usr/share/graylog/data/ssl/cacerts/cert-manager-ca.crt -noprompt\n")
		}
		if spec.TLS.HTTP.CACerts != "" {
			b.WriteString(`for FILE in /usr/share/graylog/data/ssl/cacerts/*; do keytool -importcert -keystore /usr/share/graylog/data/ssl/cacerts.jks -storepass changeit -alias "$FILE" -file "$FILE" -noprompt; done
`)
		}
	}
	b.WriteString("/docker-entrypoint.sh\n")
	return b.String()
}

func graylogContainerEnv(spec *loggingService.Graylog, def config.GraylogDefaults) []corev1.EnvVar {
	javaOpts := spec.JavaOpts
	if javaOpts == "" {
		javaOpts = def.JavaOpts
	}
	if tlsHTTPEnabled(spec) {
		javaOpts = javaOpts + " -Djavax.net.ssl.trustStore=/usr/share/graylog/data/ssl/cacerts.jks"
	}
	pathRepo := spec.PathRepo
	if pathRepo == "" {
		pathRepo = def.PathRepo
	}
	env := []corev1.EnvVar{
		{Name: "GRAYLOG_SERVER_JAVA_OPTS", Value: javaOpts},
		secretEnv("GRAYLOG_ELASTICSEARCH_HOSTS", spec.GraylogSecretName, "elasticsearchHost"),
		secretEnv("GRAYLOG_USERNAME", spec.GraylogSecretName, "user"),
		secretEnv("GRAYLOG_PASSWORD", spec.GraylogSecretName, "password"),
		{Name: "GRAYLOG_SNAPSHOT_DIRECTORY", Value: pathRepo},
	}
	if spec.S3Archive {
		env = append(env,
			secretEnv("AWS_ACCESS_KEY_ID", spec.GraylogSecretName, "awsAccessKey"),
			secretEnv("AWS_SECRET_ACCESS_KEY", spec.GraylogSecretName, "awsSecretKey"),
		)
	}
	return env
}

func secretEnv(name, secretName, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  key,
			},
		},
	}
}

func graylogContainerMounts(spec *loggingService.Graylog) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: "data", MountPath: "/usr/share/graylog/data"},
		{Name: "plugins", MountPath: "/usr/share/graylog/plugin"},
		{Name: "logsconf", MountPath: "/usr/share/graylog/data/config/log4j2.xml", SubPath: "log4j2.xml"},
		{Name: "graylogconf", MountPath: "/usr/share/graylog/data/config/graylog.conf", SubPath: "graylog.conf"},
		{Name: "nodeid", MountPath: "/usr/share/graylog/data/config/node-id", SubPath: "node-id"},
	}
	mounts = append(mounts, graylogTLSMounts(spec)...)
	return mounts
}
