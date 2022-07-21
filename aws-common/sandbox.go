package awscommon

import "github.com/pulumi/pulumi-aws/sdk/v5/go/aws"

type SandboxEnvironment struct{}

func (e SandboxEnvironment) Region() string {
	return string(aws.RegionUSEast1)
}

func (e SandboxEnvironment) ECSExecKMSKeyID() string {
	return "arn:aws:kms:us-east-1:601427279990:key/c84f93c2-a562-4a59-a326-918fbe7235c7"
}

func (e SandboxEnvironment) VPCID() string {
	return "vpc-d1aac1a8"
}
