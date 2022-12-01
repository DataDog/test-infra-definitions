package eks

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/iam"

	awsIam "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetNodeRole(e aws.Environment, name string) (*awsIam.Role, error) {
	assumeRolePolicy, err := iam.GetAWSPrincipalAssumeRole(e)
	if err != nil {
		return nil, err
	}

	return awsIam.NewRole(e.Ctx, e.Namer.ResourceName(name), &awsIam.RoleArgs{
		Name:                e.Namer.DisplayName(pulumi.String(name)),
		NamePrefix:          pulumi.StringPtr(e.Ctx.Stack() + "-" + name),
		Description:         pulumi.StringPtr("Node role for EKS Cluster: " + e.Ctx.Stack()),
		ForceDetachPolicies: pulumi.BoolPtr(true),
		ManagedPolicyArns: pulumi.ToStringArray([]string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		}),
		AssumeRolePolicy: pulumi.String(assumeRolePolicy.Json),
	}, pulumi.Provider(e.Provider))
}
