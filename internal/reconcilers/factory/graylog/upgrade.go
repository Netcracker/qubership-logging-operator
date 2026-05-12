package graylog

import (
	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/config"
	"github.com/Netcracker/qubership-logging-operator/internal/reconciler/factory/build"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// UpgradeJobName encodes the four MongoDB feature-compatibility upgrade jobs the
// controller orchestrates sequentially. Each maps to a specific MongoDB image (from
// cr.Spec.Graylog.MongoDBUpgrade for 4.0/4.2/4.4 and cr.Spec.Graylog.MongoDBImage for
// the final 5.0 step) and a setFeatureCompatibilityVersion command.
type UpgradeJobName string

const (
	UpgradeJob40 UpgradeJobName = "mongo-upgrade-job-40"
	UpgradeJob42 UpgradeJobName = "mongo-upgrade-job-42"
	UpgradeJob44 UpgradeJobName = "mongo-upgrade-job-44"
	UpgradeJob50 UpgradeJobName = "mongo-upgrade-job-50"
)

// BuildMongoUpgradeJob returns a *batchv1.Job for the named upgrade step. The 5.0 step
// uses `mongosh` (instead of `mongo`) and pulls the image from MongoDBImage on the CR;
// the others use MongoDBUpgrade.MongoDBImageXX. Pod-level fsGroup runs the upgrade as
// uid 1001 on non-OpenShift clusters.
func BuildMongoUpgradeJob(cr *loggingService.LoggingService, name UpgradeJobName, cfg *config.Defaults) *batchv1.Job {
	spec := cr.Spec.Graylog
	def := cfg.Graylog
	image, fcv, useMongosh := upgradeImageAndCommand(spec, name)
	runAsNonRoot := true
	runAsUser := int64(1001)

	var containerSC *corev1.SecurityContext
	var podSC *corev1.PodSecurityContext
	if !cr.Spec.OpenshiftDeploy {
		containerSC = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
		fsGroup := int64(1001)
		podSC = &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup}
	}

	container := build.NewContainer(MongoContainer, build.ContainerOpts{
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", upgradeScript(fcv, useMongosh)},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "mongodb", MountPath: "/data/db"},
		},
		SecurityContext: containerSC,
	})

	backoff := def.UpgradeJobBackoffLimit
	job := build.NewJob(string(name), cr.GetNamespace(), "graylog", build.JobOpts{
		BackoffLimit: &backoff,
		PodSpec: corev1.PodSpec{
			RestartPolicy:   corev1.RestartPolicyNever,
			SecurityContext: podSC,
			Volumes: []corev1.Volume{
				{Name: "mongodb", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: MongoClaim}}},
			},
			Containers: []corev1.Container{container},
		},
	})
	// Asset wrote no resource/template labels here except by post-decode hook —
	// preserve via SetLabelsForResource + a pod-template label set.
	util.SetLabelsForResource(job, util.LabelInput{
		Name:            string(name),
		Component:       "graylog",
		ComponentLabels: spec.Labels,
	}, nil)
	job.Spec.Template.Labels = util.MergeLabels(util.ResourceLabels(string(name), "graylog"), spec.Labels)
	return job
}

// upgradeImageAndCommand returns the image to run, the feature-compatibility version
// string to set, and whether to invoke `mongosh` (true on the 5.0 step) instead of
// `mongo`.
func upgradeImageAndCommand(spec *loggingService.Graylog, name UpgradeJobName) (image, fcv string, mongosh bool) {
	switch name {
	case UpgradeJob40:
		if spec.MongoDBUpgrade != nil {
			image = spec.MongoDBUpgrade.MongoDBImage40
		}
		return image, "4.0", false
	case UpgradeJob42:
		if spec.MongoDBUpgrade != nil {
			image = spec.MongoDBUpgrade.MongoDBImage42
		}
		return image, "4.2", false
	case UpgradeJob44:
		if spec.MongoDBUpgrade != nil {
			image = spec.MongoDBUpgrade.MongoDBImage44
		}
		return image, "4.4", false
	case UpgradeJob50:
		return spec.MongoDBImage, "5.0", true
	}
	return "", "", false
}

// upgradeScript builds the shell snippet each upgrade job runs: start mongod in the
// background, sleep briefly, set the feature-compatibility version, then shut down
// cleanly. mongosh replaces mongo for MongoDB 5.0+.
func upgradeScript(fcv string, mongosh bool) string {
	client := "mongo"
	if mongosh {
		client = "mongosh"
	}
	return "" +
		`mongod --wiredTigerEngineConfigString="cache_size=512M" --fork --syslog
sleep 15
` + client + ` --quiet --eval 'db.adminCommand( { setFeatureCompatibilityVersion: "` + fcv + `" } )'
mongod --shutdown
`
}
