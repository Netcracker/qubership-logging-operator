package build

import corev1 "k8s.io/api/core/v1"

// ContainerOpts captures every per-container knob the operator currently sets via YAML
// templates. Each field is consumed only when non-zero; component factories coalesce
// the CR-supplied value with config.* defaults before populating opts.
type ContainerOpts struct {
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Command         []string
	Args            []string
	Env             []corev1.EnvVar
	EnvFrom         []corev1.EnvFromSource
	Ports           []corev1.ContainerPort
	Resources       corev1.ResourceRequirements
	VolumeMounts    []corev1.VolumeMount
	LivenessProbe   *corev1.Probe
	ReadinessProbe  *corev1.Probe
	StartupProbe    *corev1.Probe
	SecurityContext *corev1.SecurityContext
	Lifecycle       *corev1.Lifecycle
}

// NewContainer assembles a corev1.Container from name + opts. Returns by value so
// callers can append into PodSpec.Containers / InitContainers directly.
func NewContainer(name string, opts ContainerOpts) corev1.Container {
	return corev1.Container{
		Name:            name,
		Image:           opts.Image,
		ImagePullPolicy: opts.ImagePullPolicy,
		Command:         opts.Command,
		Args:            opts.Args,
		Env:             opts.Env,
		EnvFrom:         opts.EnvFrom,
		Ports:           opts.Ports,
		Resources:       opts.Resources,
		VolumeMounts:    opts.VolumeMounts,
		LivenessProbe:   opts.LivenessProbe,
		ReadinessProbe:  opts.ReadinessProbe,
		StartupProbe:    opts.StartupProbe,
		SecurityContext: opts.SecurityContext,
		Lifecycle:       opts.Lifecycle,
	}
}
