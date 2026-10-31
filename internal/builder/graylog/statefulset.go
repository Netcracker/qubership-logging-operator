package graylog

import (
	"fmt"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	"github.com/Netcracker/qubership-logging-operator/internal/builder"
	"github.com/Netcracker/qubership-logging-operator/internal/defaults"
	util "github.com/Netcracker/qubership-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	dataPath         = "/usr/share/graylog/data"
	secretFilesPath  = dataPath + "/k8s-graylog-secret"
	sslPath          = dataPath + "/ssl"
	trustStoreOption = "-Djavax.net.ssl.trustStore=" + sslPath + "/cacerts.jks"
	archivesPath     = dataPath + "/archives"
	directoriesFile  = dataPath + "/config/directories.json"
	shell            = "/bin/sh"

	// Script fragments repeated in the setup script.
	mkdir        = "mkdir -p "
	chownGraylog = "chown -R 1100:1100 "

	// Volume names shared by the volumes and the volume mounts.
	authProxyConfigVolume = "graylog-auth-proxy-config"
	caCertsVolume         = "tls-cacerts"
	httpCertVolume        = "tls-cert-http"
	httpKeyVolume         = "tls-key-http"
	certManagerCAVolume   = "tls-cacerts-cert-manager"
	inputCertVolume       = "tls-cert-input"
	inputKeyVolume        = "tls-key-input"

	// Graylog runs as 1100; MongoDB, the plugin downloader, and the auth proxy run as 1001.
	graylogUser = int64(1100)
	sidecarUser = int64(1001)
)

// StatefulSet returns the Graylog StatefulSet with the MongoDB sidecar and, when it is
// installed, the auth proxy sidecar.
func StatefulSet(cr *loggingService.LoggingService, cfg *defaults.Defaults) *appsv1.StatefulSet {
	spec := cr.Spec.Graylog
	def := cfg.Graylog

	replicas := def.Replicas
	if spec.Replicas != nil {
		replicas = int32(*spec.Replicas)
	}

	containers := []corev1.Container{mongoContainer(spec, def), graylogContainer(spec, def)}
	if authProxyInstalled(spec) {
		containers = append(containers, authProxyContainer(spec, def))
	}
	podSpec := corev1.PodSpec{
		Volumes:            volumes(spec, def),
		ServiceAccountName: serviceAccountName,
		NodeSelector:       builder.NodeSelector(spec.NodeSelectorKey, spec.NodeSelectorValue),
		InitContainers:     initContainers(cr, def),
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot:        ptr.To(true),
			FSGroup:             ptr.To(sidecarUser),
			FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
			SeccompProfile:      &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
		Containers: containers,
		Affinity:   spec.Affinity,
	}
	if strings.TrimSpace(spec.PriorityClassName) != "" {
		podSpec.PriorityClassName = spec.PriorityClassName
	}

	statefulSet := builder.NewStatefulSet(statefulSetName, cr.GetNamespace(), builder.StatefulSetOpts{
		Replicas:       &replicas,
		Selector:       selector(),
		PodSpec:        podSpec,
		Annotations:    spec.Annotations,
		PodAnnotations: spec.Annotations,
	})
	util.SetLabelsForWorkload(statefulSet, &statefulSet.Spec.Template.Labels, util.LabelInput{
		Name:            statefulSetName,
		Component:       component,
		Instance:        util.GetInstanceLabel(statefulSetName, cr.GetNamespace()),
		Version:         util.GetTagFromImage(spec.DockerImage),
		Technology:      technology,
		ComponentLabels: spec.Labels,
	})
	return statefulSet
}

func initContainers(cr *loggingService.LoggingService, def defaults.GraylogDefaults) []corev1.Container {
	spec := cr.Spec.Graylog
	resources := builder.Resources(spec.InitResources, def.InitResources)
	containers := []corev1.Container{{
		Name:            "setup",
		Image:           spec.InitSetupImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: dataPath},
			{Name: "mongodb", MountPath: "/data/db"},
		},
		Command:         []string{shell, "-c", setupScript(cr.Spec.OpenshiftDeploy, httpTLSEnabled(spec))},
		SecurityContext: &corev1.SecurityContext{RunAsNonRoot: ptr.To(false)},
		Resources:       resources,
	}}
	if spec.InitContainerDockerImage != "" {
		containers = append(containers, corev1.Container{
			Name:            "download-plugins",
			Image:           spec.InitContainerDockerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts:    []corev1.VolumeMount{{Name: "plugins", MountPath: "/usr/share/graylog/plugin"}},
			Env: []corev1.EnvVar{
				{Name: "CUSTOM_PLUGINS", Value: spec.CustomPluginsPaths},
				{Name: "GRAYLOG_VERSION", Value: "5"},
			},
			SecurityContext: &corev1.SecurityContext{RunAsNonRoot: ptr.To(true), RunAsUser: ptr.To(sidecarUser)},
			Resources:       *resources.DeepCopy(),
		})
	}
	return containers
}

// setupScript prepares the MongoDB and Graylog data directories. On OpenShift the
// security context constraint picks the user, so the script opens up the permissions
// instead of relying on the group.
func setupScript(openshift, tls bool) string {
	var sb strings.Builder
	sb.WriteString("chown -R 1001:1001 /data/db\nchmod -R u=rwX,g=rwX,o= /data/db\n\n")
	if openshift {
		sb.WriteString("chmod -R 0777 " + dataPath + "/\n" +
			chownGraylog + dataPath + "\n" +
			mkdir + archivesPath + "\n" +
			chownGraylog + archivesPath + "\n" +
			mkdir + dataPath + "/config\n" +
			"touch " + directoriesFile + "\n" +
			"chmod 0666 " + directoriesFile + "\n" +
			"chown 1100:1100 " + directoriesFile + "\n" +
			"printf '{}\\n' > " + directoriesFile + "\n")
		if tls {
			sb.WriteString("\n" + mkdir + sslPath + "\n" +
				"chmod -R 0777 " + sslPath + "\n" +
				chownGraylog + sslPath + "\n")
		}
		return sb.String()
	}
	sb.WriteString(mkdir + archivesPath + "\n" +
		mkdir + dataPath + "/config\n" +
		"touch " + directoriesFile + "\n")
	if tls {
		sb.WriteString("\n" + mkdir + sslPath + "\n")
	}
	sb.WriteString("\n" + chownGraylog + dataPath + "\n" +
		"chmod -R u=rwX,g=rwX,o= " + dataPath + "\n" +
		"chmod 0660 " + directoriesFile + "\n" +
		"printf '{}\\n' > " + directoriesFile + "\n")
	return sb.String()
}

func mongoContainer(spec *loggingService.Graylog, def defaults.GraylogDefaults) corev1.Container {
	return builder.NewContainer("mongo", builder.ContainerOpts{
		Image:           spec.MongoDBImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{shell, "-c", "mongod --wiredTigerEngineConfigString=\"cache_size=512M\"\n"},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "mongodb", MountPath: "/data/db"},
			{Name: "tmp", MountPath: "/tmp"},
		},
		Ports:           []corev1.ContainerPort{{Name: "mongo", ContainerPort: mongoPort, Protocol: corev1.ProtocolTCP}},
		Resources:       builder.Resources(spec.MongoResources, def.MongoResources),
		SecurityContext: restrictedSecurityContext(sidecarUser),
	})
}

func graylogContainer(spec *loggingService.Graylog, def defaults.GraylogDefaults) corev1.Container {
	tls := httpTLSEnabled(spec)
	javaOpts := builder.First(spec.JavaOpts, def.JavaOpts)
	if tls {
		javaOpts += "\n" + trustStoreOption
	}
	env := []corev1.EnvVar{
		{Name: "GRAYLOG_SERVER_JAVA_OPTS", Value: javaOpts},
		{Name: "GRAYLOG_ELASTICSEARCH_HOSTS__FILE", Value: secretFilesPath + "/elasticsearchHost"},
		{Name: "GRAYLOG_ROOT_USERNAME__FILE", Value: secretFilesPath + "/user"},
		{Name: "GRAYLOG_ROOT_PASSWORD_SHA2__FILE", Value: secretFilesPath + "/rootPasswordSha2"},
		{Name: "GRAYLOG_SNAPSHOT_DIRECTORY", Value: spec.PathRepo},
	}
	if spec.S3Archive {
		env = append(env,
			secretEnv("AWS_ACCESS_KEY_ID", spec.GraylogSecretName, "awsAccessKey"),
			secretEnv("AWS_SECRET_ACCESS_KEY", spec.GraylogSecretName, "awsSecretKey"))
	}

	scheme := corev1.URISchemeHTTP
	if tls {
		scheme = corev1.URISchemeHTTPS
	}
	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Path:   "/",
			Port:   intstr.FromInt32(httpPort),
			Scheme: scheme,
		}},
		InitialDelaySeconds: 120,
		PeriodSeconds:       30,
		FailureThreshold:    3,
		SuccessThreshold:    1,
		TimeoutSeconds:      5,
	}

	inputPort := int32(spec.InputPort)
	securityContext := restrictedSecurityContext(graylogUser)
	// The Java binary has cap_net_bind_service=ep and does not start with an empty
	// capability bounding set. Keeping the capability also lets Graylog bind UDP port 514.
	securityContext.Capabilities.Add = []corev1.Capability{"NET_BIND_SERVICE"}

	return builder.NewContainer("graylog", builder.ContainerOpts{
		Image:           spec.DockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{shell, "-c", graylogScript(spec)},
		Env:             env,
		LivenessProbe:   probe,
		ReadinessProbe:  probe.DeepCopy(),
		VolumeMounts:    graylogVolumeMounts(spec),
		Ports: []corev1.ContainerPort{
			{Name: "graylog", ContainerPort: httpPort, Protocol: corev1.ProtocolTCP},
			{Name: "graylog-udp", ContainerPort: udpPort, Protocol: corev1.ProtocolUDP},
			{Name: fmt.Sprintf("graylog-%d", inputPort), ContainerPort: inputPort, Protocol: corev1.ProtocolTCP},
			{Name: "graylog-metrics", ContainerPort: metricsPort, Protocol: corev1.ProtocolTCP},
		},
		Resources:       builder.Resources(spec.GraylogResources, def.GraylogResources),
		SecurityContext: securityContext,
	})
}

// graylogScript starts Graylog. With HTTPS enabled, it first builds a Java trust store
// from the JDK certificates, the Graylog certificate, and the extra CA certificates.
func graylogScript(spec *loggingService.Graylog) string {
	const keytool = "keytool -importcert -keystore " + sslPath + "/cacerts.jks -storepass changeit"
	var sb strings.Builder
	sb.WriteString("if [ -f /tmp/kafka-logs/.lock ] ; then\n    rm /tmp/kafka-logs/.lock\nfi\n\n")
	if httpTLSEnabled(spec) {
		http := spec.TLS.HTTP
		sb.WriteString("cp -a \"/opt/java/openjdk/lib/security/cacerts\" \"" + sslPath + "/cacerts.jks\"\n" +
			keytool + " -alias graylog-tls-ca -file " + sslPath + "/http/tls.crt -noprompt\n")
		if generateCerts(http.GenerateCerts) {
			sb.WriteString("\n" + keytool + " -alias graylog-tls-ca-cert-manager -file " + sslPath +
				"/cacerts/cert-manager-ca.crt -noprompt\n")
		}
		sb.WriteString("\n")
		if http.CACerts != "" {
			sb.WriteString("\nfor FILE in " + sslPath + "/cacerts/*; do " + keytool +
				" -alias \"$FILE\" -file \"$FILE\" -noprompt; done\n")
		}
		sb.WriteString("\n\n")
	}
	sb.WriteString("/docker-entrypoint.sh\n")
	return sb.String()
}

func authProxyContainer(spec *loggingService.Graylog, def defaults.GraylogDefaults) corev1.Container {
	proxy := spec.AuthProxy
	htpasswdKey := ""
	if proxy.BindPasswordSecret != nil {
		htpasswdKey = proxy.BindPasswordSecret.Key
	}
	mounts := []corev1.VolumeMount{
		{Name: authProxyConfigVolume, MountPath: "/usr/src/app/config.yaml", SubPath: "config.yaml", ReadOnly: true},
		{Name: "graylog-auth-proxy-htpasswd", MountPath: "/usr/src/app/.htpasswd", SubPath: htpasswdKey, ReadOnly: true},
		{Name: "tmp", MountPath: "/tmp"},
	}
	if spec.TLS != nil && spec.TLS.HTTP != nil {
		http := spec.TLS.HTTP
		mounts = append(mounts, fileMounts(httpFiles(http, sslPath+"/http/ca.crt"))...)
		if http.CACerts != "" {
			mounts = append(mounts, readOnlyMount(caCertsVolume, sslPath+"/http/ca.crt", "ca.crt"))
		}
	}
	mounts = append(mounts, fileMounts(authProxyFiles(proxy))...)

	return builder.NewContainer(authProxyName, builder.ContainerOpts{
		Image:           proxy.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{shell, "-c", "python /usr/src/app/graylog_auth_proxy.py --config \"./config.yaml\"\n"},
		Env:             []corev1.EnvVar{{Name: "PYTHONDONTWRITEBYTECODE", Value: "1"}},
		VolumeMounts:    mounts,
		Ports: []corev1.ContainerPort{
			{Name: "proxy", ContainerPort: authProxyPort, Protocol: corev1.ProtocolTCP},
			{Name: "metrics", ContainerPort: authProxyMetrics, Protocol: corev1.ProtocolTCP},
		},
		Resources:       builder.Resources(proxy.Resources, def.AuthProxyResources),
		SecurityContext: restrictedSecurityContext(sidecarUser),
	})
}

func graylogVolumeMounts(spec *loggingService.Graylog) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: "graylog-credentials-files", MountPath: secretFilesPath, ReadOnly: true},
		{Name: "data", MountPath: dataPath},
		{Name: "plugins", MountPath: "/usr/share/graylog/plugin"},
		{Name: "plugins-merged", MountPath: "/usr/share/graylog/plugins-merged"},
		{Name: "logsconf", MountPath: dataPath + "/config/log4j2.xml", SubPath: "log4j2.xml"},
		{Name: "graylogconf", MountPath: dataPath + "/config/graylog.conf", SubPath: "graylog.conf"},
		{Name: "nodeid", MountPath: dataPath + "/config/node-id", SubPath: "node-id"},
		{Name: "tmp", MountPath: "/tmp"},
	}
	if spec.TLS == nil {
		return mounts
	}
	if http := spec.TLS.HTTP; http != nil {
		mounts = append(mounts, fileMounts(httpFiles(http, sslPath+"/cacerts/cert-manager-ca.crt"))...)
		if http.CACerts != "" {
			mounts = append(mounts, corev1.VolumeMount{Name: caCertsVolume, MountPath: sslPath + "/cacerts", ReadOnly: true})
		}
	}
	return append(mounts, fileMounts(inputFiles(spec.TLS.Input))...)
}

func volumes(spec *loggingService.Graylog, def defaults.GraylogDefaults) []corev1.Volume {
	vols := []corev1.Volume{
		pvcVolume("data", util.GraylogClaimName),
		emptyDirVolume("plugins", def.EmptyDirSize),
		emptyDirVolume("plugins-merged", def.EmptyDirSize),
		pvcVolume("mongodb", util.MongoClaimName),
		configFileVolume("logsconf", serviceName, "log4j2.xml", 0o444),
		configFileVolume("graylogconf", serviceName, "graylog.conf", 0o444),
		configFileVolume("nodeid", serviceName, "node-id", 0o666),
		emptyDirVolume("tmp", def.EmptyDirSize),
		{
			Name: "graylog-credentials-files",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName:  spec.GraylogSecretName,
				DefaultMode: ptr.To(int32(0o440)),
				Items: []corev1.KeyToPath{
					{Key: "elasticsearchHost", Path: "elasticsearchHost"},
					{Key: "user", Path: "user"},
					{Key: "rootPasswordSha2", Path: "rootPasswordSha2"},
				},
			}},
		},
	}

	if authProxyInstalled(spec) {
		proxy := spec.AuthProxy
		bindPasswordSecret := ""
		if proxy.BindPasswordSecret != nil {
			bindPasswordSecret = proxy.BindPasswordSecret.Name
		}
		vols = append(vols,
			configFileVolume(authProxyConfigVolume, authProxyConfigVolume, "config.yaml", 0o444),
			secretVolume("graylog-auth-proxy-htpasswd", bindPasswordSecret))
		vols = append(vols, fileVolumes(authProxyFiles(proxy))...)
	}

	if spec.TLS == nil {
		return vols
	}
	if http := spec.TLS.HTTP; http != nil {
		// The CA path matters only for the mounts.
		vols = append(vols, fileVolumes(httpFiles(http, ""))...)
		if http.CACerts != "" {
			vols = append(vols, secretVolume(caCertsVolume, http.CACerts))
		}
	}
	return append(vols, fileVolumes(inputFiles(spec.TLS.Input))...)
}

// secretFile is one file from a Secret, mounted at path through its own volume.
type secretFile struct {
	volume, secret, key, path string
}

// httpFiles lists the HTTPS certificate and key. Generated certificates also bring the
// cert-manager CA, which is mounted at caPath.
func httpFiles(http *loggingService.HTTPGraylogTLS, caPath string) []secretFile {
	if generateCerts(http.GenerateCerts) {
		name := http.GenerateCerts.SecretName
		return []secretFile{
			{httpCertVolume, name, "tls.crt", sslPath + "/http/tls.crt"},
			{httpKeyVolume, name, "tls.key", sslPath + "/http/tls.key"},
			{certManagerCAVolume, name, "ca.crt", caPath},
		}
	}
	return userFiles(http.Cert, http.Key, httpCertVolume, httpKeyVolume, sslPath+"/http")
}

// inputFiles lists the certificate and key of the TLS-enabled GELF input.
func inputFiles(input *loggingService.InputGraylogTLS) []secretFile {
	if input == nil {
		return nil
	}
	if generateCerts(input.GenerateCerts) {
		name := input.GenerateCerts.SecretName
		return []secretFile{
			{inputCertVolume, name, "tls.crt", sslPath + "/input/tls.crt"},
			{inputKeyVolume, name, "tls.key", sslPath + "/input/tls.key"},
		}
	}
	return userFiles(input.Cert, input.Key, inputCertVolume, inputKeyVolume, sslPath+"/input")
}

// userFiles lists the certificate and key that the user stores in Secrets. A file is
// skipped when its Secret name or key is empty.
func userFiles(cert *loggingService.Cert, key *loggingService.Key, certVolume, keyVolume, dir string) []secretFile {
	var files []secretFile
	if cert != nil && secretKeySet(cert.SecretName, cert.SecretKey) {
		files = append(files, secretFile{certVolume, cert.SecretName, cert.SecretKey, dir + "/tls.crt"})
	}
	if key != nil && secretKeySet(key.SecretName, key.SecretKey) {
		files = append(files, secretFile{keyVolume, key.SecretName, key.SecretKey, dir + "/tls.key"})
	}
	return files
}

// authProxyFiles lists the certificates the auth proxy uses to connect to LDAP.
func authProxyFiles(proxy *loggingService.AuthProxy) []secretFile {
	const dir = "/usr/share/ssl/auth"
	candidates := []secretFile{
		{"graylog-auth-proxy-ca-cert", proxy.CA.SecretName, proxy.CA.SecretKey, dir + "/ca.crt"},
		{"graylog-auth-proxy-client-cert", proxy.Cert.SecretName, proxy.Cert.SecretKey, dir + "/tls.crt"},
		{"graylog-auth-proxy-private-key", proxy.Key.SecretName, proxy.Key.SecretKey, dir + "/tls.key"},
	}
	var files []secretFile
	for _, file := range candidates {
		if secretKeySet(file.secret, file.key) {
			files = append(files, file)
		}
	}
	return files
}

func fileMounts(files []secretFile) []corev1.VolumeMount {
	mounts := make([]corev1.VolumeMount, 0, len(files))
	for _, file := range files {
		mounts = append(mounts, readOnlyMount(file.volume, file.path, file.key))
	}
	return mounts
}

func fileVolumes(files []secretFile) []corev1.Volume {
	vols := make([]corev1.Volume, 0, len(files))
	for _, file := range files {
		vols = append(vols, secretVolume(file.volume, file.secret))
	}
	return vols
}

func pvcVolume(name, claim string) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
	}}
}

func emptyDirVolume(name string, size resource.Quantity) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &size},
	}}
}

func configFileVolume(name, configMap, key string, mode int32) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: configMap},
		Items:                []corev1.KeyToPath{{Key: key, Path: key}},
		DefaultMode:          ptr.To(mode),
	}}}
}

func secretVolume(name, secret string) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
		SecretName:  secret,
		DefaultMode: ptr.To(int32(0o644)),
	}}}
}

func readOnlyMount(name, path, subPath string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: name, MountPath: path, SubPath: subPath, ReadOnly: true}
}

func secretEnv(name, secret, key string) corev1.EnvVar {
	return corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: secret},
		Key:                  key,
	}}}
}

// restrictedSecurityContext runs a container as the given user and group with a
// read-only root file system and no capabilities.
func restrictedSecurityContext(id int64) *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		RunAsUser:                ptr.To(id),
		RunAsGroup:               ptr.To(id),
	}
}

func generateCerts(gc *loggingService.GenerateCerts) bool {
	return gc != nil && gc.Enabled
}

func secretKeySet(name, key string) bool {
	return name != "" && key != ""
}
