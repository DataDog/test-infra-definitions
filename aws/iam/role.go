package iam

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
)

const (
	EC2ServicePrincipal = "ec2.amazonaws.com"
	EKSServicePrincipal = "eks.amazonaws.com"
)

func GetAWSPrincipalAssumeRole(e aws.Environment, serviceName []string) (*iam.GetPolicyDocumentResult, error) {
	return iam.GetPolicyDocument(e.Ctx, &iam.GetPolicyDocumentArgs{
		Statements: []iam.GetPolicyDocumentStatement{
			{
				Actions: []string{
					"sts:AssumeRole",
				},
				Principals: []iam.GetPolicyDocumentStatementPrincipal{
					{
						Type:        "Service",
						Identifiers: serviceName,
					},
				},
			},
		},
	}, nil, e.InvokeProviderOption())
}
