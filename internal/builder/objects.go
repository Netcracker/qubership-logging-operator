// Package builder holds the constructors that the component packages under
// internal/builder use to build Kubernetes objects from the LoggingService custom
// resource. The constructors leave metadata labels empty: the component packages set
// them with util.SetLabelsForWorkload or util.SetLabelsForResource.
package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContainerOpts lists the container fields the component builders set.
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

// NewContainer returns a container with the given name and fields.
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

// DaemonSetOpts lists the DaemonSet fields the component builders set.
type DaemonSetOpts struct {
	Selector        map[string]string
	UpdateStrategy  appsv1.DaemonSetUpdateStrategy
	MinReadySeconds int32
	PodSpec         corev1.PodSpec
	PodLabels       map[string]string
	PodAnnotations  map[string]string
	Labels          map[string]string
	Annotations     map[string]string
}

// NewDaemonSet returns a DaemonSet with the given name, namespace, and fields.
func NewDaemonSet(name, namespace string, opts DaemonSetOpts) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta:   metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
		ObjectMeta: objectMeta(name, namespace, opts.Labels, opts.Annotations),
		Spec: appsv1.DaemonSetSpec{
			Selector:        &metav1.LabelSelector{MatchLabels: opts.Selector},
			MinReadySeconds: opts.MinReadySeconds,
			UpdateStrategy:  opts.UpdateStrategy,
			Template:        podTemplate(opts.PodLabels, opts.PodAnnotations, opts.PodSpec),
		},
	}
}

// DeploymentOpts lists the Deployment fields the component builders set.
type DeploymentOpts struct {
	Replicas       *int32
	Selector       map[string]string
	Strategy       appsv1.DeploymentStrategy
	PodSpec        corev1.PodSpec
	PodLabels      map[string]string
	PodAnnotations map[string]string
	Labels         map[string]string
	Annotations    map[string]string
}

// NewDeployment returns a Deployment with the given name, namespace, and fields.
func NewDeployment(name, namespace string, opts DeploymentOpts) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: objectMeta(name, namespace, opts.Labels, opts.Annotations),
		Spec: appsv1.DeploymentSpec{
			Replicas: opts.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: opts.Selector},
			Strategy: opts.Strategy,
			Template: podTemplate(opts.PodLabels, opts.PodAnnotations, opts.PodSpec),
		},
	}
}

// StatefulSetOpts lists the StatefulSet fields the component builders set.
type StatefulSetOpts struct {
	Replicas             *int32
	ServiceName          string
	Selector             map[string]string
	PodManagementPolicy  appsv1.PodManagementPolicyType
	UpdateStrategy       appsv1.StatefulSetUpdateStrategy
	PodSpec              corev1.PodSpec
	PodLabels            map[string]string
	PodAnnotations       map[string]string
	VolumeClaimTemplates []corev1.PersistentVolumeClaim
	Labels               map[string]string
	Annotations          map[string]string
}

// NewStatefulSet returns a StatefulSet with the given name, namespace, and fields.
func NewStatefulSet(name, namespace string, opts StatefulSetOpts) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"},
		ObjectMeta: objectMeta(name, namespace, opts.Labels, opts.Annotations),
		Spec: appsv1.StatefulSetSpec{
			Replicas:             opts.Replicas,
			ServiceName:          opts.ServiceName,
			Selector:             &metav1.LabelSelector{MatchLabels: opts.Selector},
			PodManagementPolicy:  opts.PodManagementPolicy,
			UpdateStrategy:       opts.UpdateStrategy,
			Template:             podTemplate(opts.PodLabels, opts.PodAnnotations, opts.PodSpec),
			VolumeClaimTemplates: opts.VolumeClaimTemplates,
		},
	}
}

// ServiceOpts lists the Service fields the component builders set.
type ServiceOpts struct {
	Type           corev1.ServiceType
	ClusterIP      string
	Ports          []corev1.ServicePort
	Selector       map[string]string
	IPFamilies     []corev1.IPFamily
	IPFamilyPolicy *corev1.IPFamilyPolicy
	Labels         map[string]string
	Annotations    map[string]string
}

// NewService returns a Service with the given name, namespace, and fields.
func NewService(name, namespace string, opts ServiceOpts) *corev1.Service {
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: objectMeta(name, namespace, opts.Labels, opts.Annotations),
		Spec: corev1.ServiceSpec{
			Type:           opts.Type,
			ClusterIP:      opts.ClusterIP,
			Ports:          opts.Ports,
			Selector:       opts.Selector,
			IPFamilies:     opts.IPFamilies,
			IPFamilyPolicy: opts.IPFamilyPolicy,
		},
	}
}

// NewServiceAccount returns a ServiceAccount with the given name and namespace.
func NewServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{Kind: "ServiceAccount", APIVersion: "v1"},
		ObjectMeta: objectMeta(name, namespace, nil, nil),
	}
}

// NodeSelector returns a one-entry node selector, or nil when the key or the value is empty.
func NodeSelector(key, value string) map[string]string {
	if key == "" || value == "" {
		return nil
	}
	return map[string]string{key: value}
}

func objectMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels, Annotations: annotations}
}

func podTemplate(labels, annotations map[string]string, spec corev1.PodSpec) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
		Spec:       spec,
	}
}
