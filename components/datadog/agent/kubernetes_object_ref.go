package agent

import (
	"encoding/json"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
)

LabelSelectorsRef struct {
	Name pulumi.String `pulumi:"name"`
	Value pulumi.String `pulumi:"value"`
}
type KubernetesObjRefOutput struct {
	components.JSONImporter

	Namespace      string `json:"namespace"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	AppVersion     string `json:"installAppVersion"`
	Version        string `json:"installVersion"`
	LabelSelectors string `json:"labelSelectors"`
}

type KubernetesObjectRef struct {
	pulumi.ResourceState
	components.Component

	Namespace      pulumi.String       `pulumi:"namespace"`
	Name           pulumi.String       `pulumi:"name"`
	Kind           pulumi.String       `pulumi:"kind"`
	AppVersion     pulumi.StringOutput `pulumi:"installAppVersion"`
	Version        pulumi.StringOutput `pulumi:"installVersion"`
	LabelSelectors pulumi.String       `pulumi:"labelSelectors"`
}

func NewKubernetesObjRef(e config.Env, name string, namespace string, kind string, appVersion pulumi.StringOutput, version pulumi.StringOutput, labelSelectors map[string]string) (*KubernetesObjectRef, error) {
	return components.NewComponent(e, name, func(comp *KubernetesObjectRef) error {
		comp.Name = pulumi.String(name)
		comp.Namespace = pulumi.String(namespace)
		comp.Kind = pulumi.String(kind)
		comp.AppVersion = appVersion
		comp.Version = version

		labelSelectorsJSONStr, err := json.Marshal(labelSelectors)
		if err != nil {
			return err
		}
		comp.LabelSelectors = pulumi.String(labelSelectorsJSONStr)

		return nil
	})
}

func (h *KubernetesObjectRef) Export(ctx *pulumi.Context, out *KubernetesObjRefOutput) error {
	return components.Export(ctx, h, out)
}
