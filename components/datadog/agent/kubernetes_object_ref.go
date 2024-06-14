package agent

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
)

type KubernetesObjRefOutput struct {
	components.JSONImporter

	Namespace     pulumi.String       `json:"namespace"`
	Name          pulumi.String       `json:"name"`
	Kind          pulumi.String       `json:"kind"`
	AppVersion    pulumi.StringOutput `json:"installAppVersion"`
	Version       pulumi.StringOutput `json:"installVersion"`
	LabelSelector pulumi.Output       `json:"labelSelectors"`
}

type KubernetesObjectRef struct {
	pulumi.ResourceState
	components.Component

	Namespace     pulumi.String       `pulumi:"namespace"`
	Name          pulumi.String       `pulumi:"name"`
	Kind          pulumi.String       `pulumi:"kind"`
	AppVersion    pulumi.StringOutput `pulumi:"installAppVersion"`
	Version       pulumi.StringOutput `pulumi:"installVersion"`
	LabelSelector pulumi.Output       `pulumi:"labelSelectors"`
}

func NewKubernetesObjRef(e config.Env, name string, namespace string, kind string, appVersion pulumi.StringOutput, version pulumi.StringOutput, labelSelectors pulumi.Map) (*KubernetesObjectRef, error) {
	return components.NewComponent(e, name, func(comp *KubernetesObjectRef) error {
		comp.Name = pulumi.String(name)
		comp.Namespace = pulumi.String(namespace)
		comp.Kind = pulumi.String(kind)
		comp.AppVersion = appVersion
		comp.Version = version
		comp.LabelSelector = labelSelectors.ToMapOutput()

		return nil
	})
}

func (h *KubernetesObjectRef) Export(ctx *pulumi.Context, out *KubernetesObjRefOutput) error {
	return components.Export(ctx, h, out)
}
