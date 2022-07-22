package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type SandboxEnvironment struct{}

func NewSandboxEnvironment(config auto.ConfigMap) Environment {
	env := SandboxEnvironment{}
	env.fillConfigMap(config)

	return env
}

func (e SandboxEnvironment) fillConfigMap(config auto.ConfigMap) {
	if config == nil {
		return
	}

	config["aws:region"] = auto.ConfigValue{
		Value: e.Region(),
	}
}

func (e SandboxEnvironment) Region() string {
	return string(aws.RegionUSEast1)
}

func (e SandboxEnvironment) ECSExecKMSKeyID() string {
	return "arn:aws:kms:us-east-1:601427279990:key/c84f93c2-a562-4a59-a326-918fbe7235c7"
}

func (e SandboxEnvironment) ECSTaskExecutionRole() string {
	return "arn:aws:iam::601427279990:role/ecsExecTaskExecutionRole"
}

func (e SandboxEnvironment) ECSTaskRole() string {
	return "arn:aws:iam::601427279990:role/ecsExecTaskRole"
}

func (e SandboxEnvironment) APIKeySSMParamName() string {
	return "agent.ci.dev.apikey"
}

func (e SandboxEnvironment) VPCID() string {
	return "vpc-d1aac1a8"
}

func (e SandboxEnvironment) AssignPublicIP() bool {
	return false
}

func (e SandboxEnvironment) DefaultSubnet() string {
	return "subnet-b89e00e2"
}

func (e SandboxEnvironment) DefaultSecurityGroups() []string {
	return []string{
		"sg-46506837", // Default
		"sg-7fedd80a", // appgate-service
	}
}
