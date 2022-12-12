package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
)

const (
	sandboxEnv = "aws/sandbox"
)

type environmentDefault struct {
	aws     awsProvider
	ddInfra ddInfra
}

type awsProvider struct {
	region string
}

type ddInfra struct {
	defaultVPCID               string
	defaultSubnets             []string
	defaultSecurityGroups      []string
	defaultInstanceType        string
	defaultARMInstanceType     string
	defaultInstanceStorageSize int

	ecs ddInfraECS
	eks ddInfraEKS
}

type ddInfraECS struct {
	execKMSKeyID                  string
	taskExecutionRole             string
	taskRole                      string
	instanceProfile               string
	serviceAllocatePublicIP       bool
	fargateCapacityProvider       bool
	linuxECSOptimizedNodeGroup    bool
	linuxECSOptimizedARMNodeGroup bool
	linuxBottlerocketNodeGroup    bool
	windowsLTSCNodeGroup          bool
}

type ddInfraEKS struct {
	allowedInboundSecurityGroups []string
	fargateNamespace             string
	linuxNodeGroup               bool
	linuxARMNodeGroup            bool
	linuxBottlerocketNodeGroup   bool
	windowsLTSCNodeGroup         bool
}

func getEnvironmentDefault(envName string) environmentDefault {
	switch envName {
	case sandboxEnv:
		return sandboxDefault()
	default:
		panic("Unknown environment: " + envName)
	}
}

func sandboxDefault() environmentDefault {
	return environmentDefault{
		aws: awsProvider{
			region: string(aws.RegionUSEast1),
		},
		ddInfra: ddInfra{
			defaultVPCID:               "vpc-d1aac1a8",
			defaultSubnets:             []string{"subnet-b89e00e2", "subnet-8ee8b1c6", "subnet-3f5db45b"},
			defaultSecurityGroups:      []string{"sg-46506837", "sg-7fedd80a"},
			defaultInstanceType:        "t3.xlarge",
			defaultARMInstanceType:     "t4g.xlarge",
			defaultInstanceStorageSize: 8,

			ecs: ddInfraECS{
				execKMSKeyID:               "arn:aws:kms:us-east-1:601427279990:key/c84f93c2-a562-4a59-a326-918fbe7235c7",
				taskExecutionRole:          "arn:aws:iam::601427279990:role/ecsExecTaskExecutionRole",
				taskRole:                   "arn:aws:iam::601427279990:role/ecsExecTaskRole",
				instanceProfile:            "arn:aws:iam::601427279990:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:    false,
				fargateCapacityProvider:    true,
				linuxECSOptimizedNodeGroup: true,
				linuxBottlerocketNodeGroup: true,
				windowsLTSCNodeGroup:       true,
			},

			eks: ddInfraEKS{
				allowedInboundSecurityGroups: []string{"sg-46506837", "sg-b9e2ebcb"},
				fargateNamespace:             "fargate",
				linuxNodeGroup:               true,
				linuxARMNodeGroup:            true,
				linuxBottlerocketNodeGroup:   true,
				windowsLTSCNodeGroup:         true,
			},
		},
	}
}
