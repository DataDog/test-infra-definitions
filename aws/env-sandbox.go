package aws

import (
	"github.com/DataDog/test-infra-definitions/common/config"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
)

func GetSandboxEnvironmentConfig() config.EnvironmentConfig {
	return config.EnvironmentConfig{
		Config: config.Config{
			AWS: config.AWS{
				Region: string(aws.RegionUSEast1),
			},
			DDInfra: config.DDInfra{
				AWS: config.AWSDDInfra{
					DefaultVPCID:          "vpc-d1aac1a8",
					DefaultSubnets:        []string{"subnet-b89e00e2"},
					DefaultSecurityGroups: []string{"sg-46506837", "sg-7fedd80a"},
					DefaultInstanceType:   "t3.xlarge",
				},
				ECS: config.ECS{
					ExecKMSKeyID:               "arn:aws:kms:us-east-1:601427279990:key/c84f93c2-a562-4a59-a326-918fbe7235c7",
					TaskExecutionRole:          "arn:aws:iam::601427279990:role/ecsExecTaskExecutionRole",
					TaskRole:                   "arn:aws:iam::601427279990:role/ecsExecTaskRole",
					ServiceAllocatePublicIP:    false,
					FargateCapacityProvider:    true,
					LinuxECSOptimizedNodeGroup: true,
					LinuxBottlerocketNodeGroup: true,
					WindowsLTSCNodeGroup:       true,
				},
			},
		},
	}
}
