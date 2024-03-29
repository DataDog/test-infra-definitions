package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
)

const (
	sandboxEnv       = "aws/sandbox"
	agentSandboxEnv  = "aws/agent-sandbox"
	agentQAEnv       = "aws/agent-qa"
	tsePlaygroundEnv = "aws/tse-playground"
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
	defaultInstanceProfileName string
	defaultARMInstanceType     string
	defaultInstanceStorageSize int
	defaultShutdownBehavior    string
	defaultInternalRegistry    string

	ecs ddInfraECS
	eks ddInfraEKS
}

type ddInfraECS struct {
	execKMSKeyID                  string
	fargateFakeintakeClusterArn   string
	fakeintakeLBListenerArn       string
	fakeintakeLBBaseHostHeader    string
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
	allowedInboundPrefixList     []string
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
	case agentSandboxEnv:
		return agentSandboxDefault()
	case agentQAEnv:
		return agentQADefault()
	case tsePlaygroundEnv:
		return tsePlaygroundDefault()
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
			defaultSecurityGroups:      []string{"sg-46506837", "sg-7fedd80a", "sg-0e952e295ab41e748"},
			defaultInstanceType:        "t3.medium",
			defaultInstanceProfileName: "ec2InstanceRole",
			defaultARMInstanceType:     "t4g.medium",
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",
			defaultInternalRegistry:    "669783387624.dkr.ecr.us-east-1.amazonaws.com",

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

func agentSandboxDefault() environmentDefault {
	return environmentDefault{
		aws: awsProvider{
			region: string(aws.RegionUSEast1),
		},
		ddInfra: ddInfra{
			defaultVPCID:               "vpc-029c0faf8f49dee8d",
			defaultSubnets:             []string{"subnet-0a15f3482cd3f9820", "subnet-091570395d476e9ce", "subnet-003831c49a10df3dd"},
			defaultSecurityGroups:      []string{"sg-038231b976eb13d44", "sg-05466e7ce253d21b1"},
			defaultInstanceType:        "t3.medium",
			defaultInstanceProfileName: "ec2InstanceRole",
			defaultARMInstanceType:     "t4g.medium",
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",
			defaultInternalRegistry:    "669783387624.dkr.ecr.us-east-1.amazonaws.com",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:376334461865:key/1d1fe533-a4f1-44ee-99ec-225b44fcb9ed",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:376334461865:cluster/fakeintake-ecs",
				fakeintakeLBListenerArn:     "arn:aws:elasticloadbalancing:us-east-1:376334461865:listener/app/fakeintake/3bbebae6506eb8cb/eea87c947a30f106",
				fakeintakeLBBaseHostHeader:  ".lb1.fi.sandbox.dda-testing.com",
				taskExecutionRole:           "arn:aws:iam::376334461865:role/ecsTaskExecutionRole",
				taskRole:                    "arn:aws:iam::376334461865:role/ecsTaskRole",
				instanceProfile:             "arn:aws:iam::376334461865:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:     false,
				fargateCapacityProvider:     true,
				linuxECSOptimizedNodeGroup:  true,
				linuxBottlerocketNodeGroup:  true,
				windowsLTSCNodeGroup:        true,
			},

			eks: ddInfraEKS{
				allowedInboundSecurityGroups: []string{"sg-038231b976eb13d44", "sg-0d82a3ae7646ca5f4"},
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
			defaultSecurityGroups:      []string{"sg-05e9573fcc582f22c", "sg-0498c960a173dff1e"},
			defaultInstanceType:        "t3.medium",
			defaultInstanceProfileName: "ec2InstanceRole",
			defaultARMInstanceType:     "t4g.medium",
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",
			defaultInternalRegistry:    "669783387624.dkr.ecr.us-east-1.amazonaws.com",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:669783387624:key/384373bc-6d99-4d68-84b5-b76b756b0af3",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:669783387624:cluster/fakeintake-ecs",
				fakeintakeLBListenerArn:     "arn:aws:elasticloadbalancing:us-east-1:669783387624:listener/app/fakeintake/de7956e70776e471/ddfa738893c2dc0e",
				fakeintakeLBBaseHostHeader:  ".lb1.fi.qa.dda-testing.com",
				taskExecutionRole:           "arn:aws:iam::669783387624:role/ecsTaskExecutionRole",
				taskRole:                    "arn:aws:iam::669783387624:role/ecsTaskRole",
				instanceProfile:             "arn:aws:iam::669783387624:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:     false,
				fargateCapacityProvider:     true,
				linuxECSOptimizedNodeGroup:  true,
				linuxBottlerocketNodeGroup:  true,
				windowsLTSCNodeGroup:        true,
			},

			eks: ddInfraEKS{
				allowedInboundSecurityGroups: []string{"sg-05e9573fcc582f22c", "sg-070023ab71cadf760"},
				allowedInboundPrefixList:     []string{"pl-0a698837099ae16f4"},
				fargateNamespace:             "fargate",
				linuxNodeGroup:               true,
				linuxARMNodeGroup:            true,
				linuxBottlerocketNodeGroup:   true,
				windowsLTSCNodeGroup:         true,
			},
		},
	}
}

func tsePlaygroundDefault() environmentDefault {
	return environmentDefault{
		aws: awsProvider{
			region: string(aws.RegionUSEast1),
		},
		ddInfra: ddInfra{
			defaultVPCID:               "vpc-0570ac09560a97693",
			defaultSubnets:             []string{"subnet-0ec4b9823cf352b95", "subnet-0e9c45e996754e357", "subnet-070e1a6c79f6bc499"},
			defaultSecurityGroups:      []string{"sg-091a00b0944f04fd2", "sg-073f15b823d4bb39a", "sg-0a3ec6b0ee295e826"},
			defaultInstanceType:        "t3.medium",
			defaultARMInstanceType:     "t4g.medium",
			defaultInstanceStorageSize: 200,
			defaultShutdownBehavior:    "stop",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:570690476889:key/f1694e5a-bb52-42a7-b414-dfd34fbd6759",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:570690476889:cluster/fakeintake-ecs",
				taskExecutionRole:           "arn:aws:iam::570690476889:role/ecsExecTaskExecutionRole",
				taskRole:                    "arn:aws:iam::570690476889:role/ecsExecTaskRole",
				instanceProfile:             "arn:aws:iam::570690476889:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:     false,
				fargateCapacityProvider:     true,
				linuxECSOptimizedNodeGroup:  true,
				linuxBottlerocketNodeGroup:  true,
				windowsLTSCNodeGroup:        true,
			},

			eks: ddInfraEKS{
				allowedInboundSecurityGroups: []string{"sg-091a00b0944f04fd2", "sg-0a3ec6b0ee295e826"},
				fargateNamespace:             "fargate",
				linuxNodeGroup:               true,
				linuxARMNodeGroup:            true,
				linuxBottlerocketNodeGroup:   true,
				windowsLTSCNodeGroup:         true,
			},
		},
	}
}
