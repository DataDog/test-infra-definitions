package operator

import (
	compkubernetes "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/components"
)

// OperatorOutput is used to import the Operator component
type OperatorOutput struct {
	components.JSONImporter

	Operator compkubernetes.KubernetesObjRefOutput `pulumi:"operator"`
}

// Operator represents an Operator installation
type Operator struct {
	pulumi.ResourceState
	components.Component

	Operator *compkubernetes.KubernetesObjectRef `pulumi:"operator"`
}

func (o *Operator) Export(ctx *pulumi.Context, out *OperatorOutput) error {
	return components.Export(ctx, o, out)
}
