package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
)

const (
	sandboxEnv = "aws/sandbox"
	agentQAEnv = "aws/agent-qa"
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
	defaultShutdownBehavior    string

	ecs ddInfraECS
	eks ddInfraEKS
}

type ddInfraECS struct {
	execKMSKeyID                  string
	fargateFakeintakeClusterArn   string
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
	case agentQAEnv:
		return agentQADefault()
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
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:601427279990:key/c84f93c2-a562-4a59-a326-918fbe7235c7",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:601427279990:cluster/fakeintake-ecs",
				taskExecutionRole:           "arn:aws:iam::601427279990:role/ecsExecTaskExecutionRole",
				taskRole:                    "arn:aws:iam::601427279990:role/ecsExecTaskRole",
				instanceProfile:             "arn:aws:iam::601427279990:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:     false,
				fargateCapacityProvider:     true,
				linuxECSOptimizedNodeGroup:  true,
				linuxBottlerocketNodeGroup:  true,
				windowsLTSCNodeGroup:        true,
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

func agentQADefault() environmentDefault {
	return environmentDefault{
		aws: awsProvider{
			region: string(aws.RegionUSEast1),
		},
		ddInfra: ddInfra{
			defaultVPCID:               "vpc-0097b9307ec2c8139",
			defaultSubnets:             []string{"subnet-0f1ca3e929eb3fb8b", "subnet-03061a1647c63c3c3", "subnet-071213aedb0e1ae54"},
			defaultSecurityGroups:      []string{"sg-05e9573fcc582f22c"},
			defaultInstanceType:        "t3.xlarge",
			defaultARMInstanceType:     "t4g.xlarge",
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",

			ecs: ddInfraECS{
				execKMSKeyID: "arn:aws:kms:us-east-1:669783387624:key/384373bc-6d99-4d68-84b5-b76b756b0af3",
				// TODO add dedicated fargate cluster to agent/qa and add it here
				// fargateFakeintakeClusterArn: "TODO",
				taskExecutionRole:          "arn:aws:iam::669783387624:role/ecsTaskExecutionRole",
				taskRole:                   "arn:aws:iam::669783387624:role/ecsTaskRole",
				instanceProfile:            "arn:aws:iam::669783387624:instance-profile/ecsInstanceRole-profile",
				serviceAllocatePublicIP:    false,
				fargateCapacityProvider:    true,
				linuxECSOptimizedNodeGroup: true,
				linuxBottlerocketNodeGroup: true,
				windowsLTSCNodeGroup:       true,
			},

			eks: ddInfraEKS{
				allowedInboundSecurityGroups: []string{"sg-05e9573fcc582f22c"},
				fargateNamespace:             "fargate",
				linuxNodeGroup:               true,
				linuxARMNodeGroup:            true,
				linuxBottlerocketNodeGroup:   true,
				windowsLTSCNodeGroup:         true,
			},
		},
	}
}
