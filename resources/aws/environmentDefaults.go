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
	region  string
	profile string
}

type FakeintakeLBConfig struct {
	listenerArn string
	baseHost    string
}

type ddInfra struct {
	defaultVPCID                   string
	defaultSubnets                 []string
	defaultSecurityGroups          []string
	defaultInstanceType            string
	defaultInstanceProfileName     string
	defaultARMInstanceType         string
	defaultInstanceStorageSize     int
	defaultShutdownBehavior        string
	defaultInternalRegistry        string
	defaultInternalDockerhubMirror string

	ecs ddInfraECS
	eks ddInfraEKS
}

type ddInfraECS struct {
	execKMSKeyID                  string
	fargateFakeintakeClusterArn   string
	defaultFakeintakeLBs          []FakeintakeLBConfig
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
	podSubnets                   []DDInfraEKSPodSubnets
	allowedInboundSecurityGroups []string
	allowedInboundPrefixList     []string
	fargateNamespace             string
	linuxNodeGroup               bool
	linuxARMNodeGroup            bool
	linuxBottlerocketNodeGroup   bool
	windowsLTSCNodeGroup         bool
}

type DDInfraEKSPodSubnets struct {
	AZ       string `json:"az"`
	SubnetID string `json:"subnet"`
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
			region:  string(aws.RegionUSEast1),
			profile: "exec-sso-sandbox-account-admin",
		},
		ddInfra: ddInfra{
			defaultVPCID:                   "vpc-d1aac1a8",
			defaultSubnets:                 []string{"subnet-b89e00e2", "subnet-8ee8b1c6", "subnet-3f5db45b"},
			defaultSecurityGroups:          []string{"sg-46506837", "sg-7fedd80a", "sg-0e952e295ab41e748"},
			defaultInstanceType:            "t3.medium",
			defaultInstanceProfileName:     "ec2InstanceRole",
			defaultARMInstanceType:         "t4g.medium",
			defaultInstanceStorageSize:     200,
			defaultShutdownBehavior:        "stop",
			defaultInternalRegistry:        "669783387624.dkr.ecr.us-east-1.amazonaws.com",
			defaultInternalDockerhubMirror: "669783387624.dkr.ecr.us-east-1.amazonaws.com/dockerhub",

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
			region:  string(aws.RegionUSEast1),
			profile: "exec-sso-agent-sandbox-account-admin",
		},
		ddInfra: ddInfra{
			defaultVPCID:                   "vpc-029c0faf8f49dee8d",
			defaultSubnets:                 []string{"subnet-0a15f3482cd3f9820", "subnet-091570395d476e9ce", "subnet-003831c49a10df3dd"},
			defaultSecurityGroups:          []string{"sg-038231b976eb13d44", "sg-05466e7ce253d21b1"},
			defaultInstanceType:            "t3.medium",
			defaultInstanceProfileName:     "ec2InstanceRole",
			defaultARMInstanceType:         "t4g.medium",
			defaultInstanceStorageSize:     200,
			defaultShutdownBehavior:        "stop",
			defaultInternalRegistry:        "669783387624.dkr.ecr.us-east-1.amazonaws.com",
			defaultInternalDockerhubMirror: "669783387624.dkr.ecr.us-east-1.amazonaws.com/dockerhub",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:376334461865:key/1d1fe533-a4f1-44ee-99ec-225b44fcb9ed",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:376334461865:cluster/fakeintake-ecs",
				defaultFakeintakeLBs: []FakeintakeLBConfig{
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:376334461865:listener/app/fakeintake/3bbebae6506eb8cb/eea87c947a30f106", baseHost: ".lb1.fi.sandbox.dda-testing.com"},
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:376334461865:listener/app/fakeintake2/e514320b44979d84/3df6c797d971c13b", baseHost: ".lb2.fi.sandbox.dda-testing.com"},
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:376334461865:listener/app/fakeintake3/1af15fb150ca4eb4/88e1d12c35e7aba0", baseHost: ".lb3.fi.sandbox.dda-testing.com"},
				},
				taskExecutionRole:          "arn:aws:iam::376334461865:role/ecsTaskExecutionRole",
				taskRole:                   "arn:aws:iam::376334461865:role/ecsTaskRole",
				instanceProfile:            "arn:aws:iam::376334461865:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:    false,
				fargateCapacityProvider:    true,
				linuxECSOptimizedNodeGroup: true,
				linuxBottlerocketNodeGroup: true,
				windowsLTSCNodeGroup:       true,
			},

			eks: ddInfraEKS{
				podSubnets: []DDInfraEKSPodSubnets{
					{
						AZ:       "us-east-1a",
						SubnetID: "subnet-0159c891fdb0ab50b",
					},
					{
						AZ:       "us-east-1b",
						SubnetID: "subnet-01cb353bec8f2b3e6",
					},
					{
						AZ:       "us-east-1d",
						SubnetID: "subnet-0ba7fbd4fed03bbdd",
					},
				},
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
			region:  string(aws.RegionUSEast1),
			profile: "exec-sso-agent-qa-account-admin",
		},
		ddInfra: ddInfra{
			defaultVPCID:                   "vpc-0097b9307ec2c8139",
			defaultSubnets:                 []string{"subnet-0d8bd689da421970c", "subnet-06eecbdafc2dac21e", "subnet-09540c6dec9c38018"},
			defaultSecurityGroups:          []string{"sg-05e9573fcc582f22c", "sg-0498c960a173dff1e"},
			defaultInstanceType:            "t3.medium",
			defaultInstanceProfileName:     "ec2InstanceRole",
			defaultARMInstanceType:         "t4g.medium",
			defaultInstanceStorageSize:     200,
			defaultShutdownBehavior:        "stop",
			defaultInternalRegistry:        "669783387624.dkr.ecr.us-east-1.amazonaws.com",
			defaultInternalDockerhubMirror: "669783387624.dkr.ecr.us-east-1.amazonaws.com/dockerhub",

			ecs: ddInfraECS{
				execKMSKeyID:                "arn:aws:kms:us-east-1:669783387624:key/384373bc-6d99-4d68-84b5-b76b756b0af3",
				fargateFakeintakeClusterArn: "arn:aws:ecs:us-east-1:669783387624:cluster/fakeintake-ecs",
				defaultFakeintakeLBs: []FakeintakeLBConfig{
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:669783387624:listener/app/fakeintake/de7956e70776e471/ddfa738893c2dc0e", baseHost: ".lb1.fi.qa.dda-testing.com"},
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:669783387624:listener/app/fakeintake2/d59e26c0a29d8567/52a83f7da0f000ee", baseHost: ".lb2.fi.qa.dda-testing.com"},
					{listenerArn: "arn:aws:elasticloadbalancing:us-east-1:669783387624:listener/app/fakeintake3/f90da6a0eaf5638d/647ea5aff700de43", baseHost: ".lb3.fi.qa.dda-testing.com"},
				},
				taskExecutionRole:          "arn:aws:iam::669783387624:role/ecsTaskExecutionRole",
				taskRole:                   "arn:aws:iam::669783387624:role/ecsTaskRole",
				instanceProfile:            "arn:aws:iam::669783387624:instance-profile/ecsInstanceRole",
				serviceAllocatePublicIP:    false,
				fargateCapacityProvider:    true,
				linuxECSOptimizedNodeGroup: true,
				linuxBottlerocketNodeGroup: true,
				windowsLTSCNodeGroup:       true,
			},

			eks: ddInfraEKS{
				podSubnets: []DDInfraEKSPodSubnets{
					{
						AZ:       "us-east-1a",
						SubnetID: "subnet-02cef8d896085b24b",
					},
					{
						AZ:       "us-east-1b",
						SubnetID: "subnet-0950e55ed25f3bdc0",
					},
					{
						AZ:       "us-east-1d",
						SubnetID: "subnet-0190651c83b3ebbbe",
					},
				},
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
			region:  string(aws.RegionUSEast1),
			profile: "exec-sso-tse-playground-account-admin",
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
