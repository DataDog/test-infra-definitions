package k8s

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DeploymentModifier is a function that operates on a DeploymentArgs struct
type DeploymentModifier func(args *appsv1.DeploymentArgs)

// rawMapToStringMap converts map[string]string
func rawMapToStringMap(raw map[string]string) pulumi.StringMap {
	result := pulumi.StringMap{}
	for k, v := range raw {
		result[k] = pulumi.String(v)
	}
	return result
}

// mergeStringMaps merges two pulumi.StringMaps into one
func mergeStringMaps(left, right pulumi.StringMap) pulumi.StringMap {
	// buffer
	merged := pulumi.StringMap{}
	// get everything from left map
	for k, v := range left {
		merged[k] = v
	}
	// get everything from right map, this will overwrite if
	// keys are duplicated
	for k, v := range right {
		merged[k] = v
	}

	return merged
}

// ensureDeploymentPodTemplateSpec performs nil and type checks
// on a DeploymentArgs struct and returns PodSpecArgs
func ensureDeploymentPodTemplateSpec(d *appsv1.DeploymentArgs) (*corev1.PodSpecArgs, bool) {
	// nil check spec
	deploymentSpecPtr := d.Spec
	if deploymentSpecPtr == nil {
		d.Spec = &appsv1.DeploymentSpecArgs{}
	}
	// type check spec
	spec, ok := deploymentSpecPtr.(*appsv1.DeploymentSpecArgs)
	if !ok {
		return nil, false
	}

	// nil check spec.Template
	podTemplatePtr := spec.Template
	if podTemplatePtr == nil {
		spec.Template = &corev1.PodTemplateSpecArgs{}
	}
	// type check spec.Template
	podTemplate, ok := podTemplatePtr.(*corev1.PodTemplateSpecArgs)
	if !ok {
		return nil, false
	}

	// nil check spec.Template.Spec
	podTemplateSpecPtr := podTemplate.Spec
	if podTemplateSpecPtr == nil {
		podTemplate.Spec = &corev1.PodSpecArgs{}
	}

	// type check spec.Template.Spec
	podTemplateSpec, ok := podTemplateSpecPtr.(*corev1.PodSpecArgs)
	if !ok {
		return nil, false
	}

	return podTemplateSpec, true
}

// WithRuntimeClass sets a deployment's RuntimeClassName
func WithRuntimeClass(rtc string) DeploymentModifier {
	return func(d *appsv1.DeploymentArgs) {
		if podTemplateSpec, ok := ensureDeploymentPodTemplateSpec(d); ok {
			podTemplateSpec.RuntimeClassName = runtimeClassToPulumi(rtc)
		}
	}
}

// WithServiceAccount sets a deployment's ServiceAccount
func WithServiceAccount(serviceAccount *corev1.ServiceAccount) DeploymentModifier {
	return func(d *appsv1.DeploymentArgs) {
		if podTemplateSpec, ok := ensureDeploymentPodTemplateSpec(d); ok {
			podTemplateSpec.ServiceAccount = serviceAccount.Metadata.Name()
		}
	}
}

// WithLabels appends/ovewrites a Deployment template's labels
func WithLabels(labels map[string]string) DeploymentModifier {
	return func(m *appsv1.DeploymentArgs) {
		// nil check spec
		specPtr := m.Spec
		if specPtr == nil {
			m.Spec = &appsv1.DeploymentSpecArgs{}
		}
		// type check spec
		spec, ok := specPtr.(*appsv1.DeploymentSpecArgs)
		if !ok {
			return
		}

		// nil check spec.Template
		templatePtr := spec.Template
		if templatePtr == nil {
			spec.Template = &corev1.PodTemplateSpecArgs{}
		}
		// type check spec.Template
		template, ok := templatePtr.(*corev1.PodTemplateSpecArgs)
		if !ok {
			return
		}

		// nil check template.Metadata
		metadataPtr := template.Metadata
		if metadataPtr == nil {
			template.Metadata = &metav1.ObjectMetaArgs{}
		}

		// assign labels
		if metadata, ok := metadataPtr.(*metav1.ObjectMetaArgs); ok {
			// If labels is nil initialize it
			if metadata.Labels == nil {
				metadata.Labels = pulumi.StringMap{}
			}
			// merge the existing labels with new ones
			merged := mergeStringMaps(metadata.Labels.(pulumi.StringMap), rawMapToStringMap(labels))
			// reassign
			metadata.Labels = merged
		}
	}
}
