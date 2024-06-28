package operator

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/operatorparams"
)

// OperatorOutput is used to import the Operator component
type OperatorOutput struct {
	components.JSONImporter
}

// Operator represents an Operator installation
type Operator struct {
	pulumi.ResourceState
	components.Component
}

func (h *Operator) Export(ctx *pulumi.Context, out *OperatorOutput) error {
	return components.Export(ctx, h, out)
}

func NewOperator(e config.Env, resourceName string, kubeProvider *kubernetes.Provider, options ...operatorparams.Option) (*Operator, error) {
	return components.NewComponent(e, resourceName, func(comp *Operator) error {
		params, err := operatorparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		_, err = NewHelmInstallation(e, HelmInstallationArgs{
			KubeProvider:          kubeProvider,
			Namespace:             params.Namespace,
			ValuesYAML:            params.HelmValues,
			OperatorFullImagePath: params.OperatorFullImagePath,
		}, params.PulumiResourceOptions...)
		if err != nil {
			return err
		}

		return nil
	})
}
