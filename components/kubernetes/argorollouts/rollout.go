package argorollouts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func RolloutFromDeployment(ctx *pulumi.Context, name string, deployment *v1.DeploymentArgs, opts ...pulumi.ResourceOption) error {
	specOutput := deployment.Spec.ToDeploymentSpecPtrOutput()
	_, err := apiextensions.NewCustomResource(ctx, name, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Rollout"),
		Metadata:   deployment.Metadata,
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"strategy": pulumi.Map{
					"canary": pulumi.Map{},
				},
				"replicas": specOutput.Replicas(),
				"selector": specOutput.Selector(),
				"template": specOutput.Template(),
			},
		},
	}, opts...)
	if err != nil {
		return err
	}
	return nil
}
