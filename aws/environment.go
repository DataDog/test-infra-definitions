package aws

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	awsConfigNamespace = "aws"

	awsRegionParamName = "region"

	// AWS Infra
	ddinfraDefaultVPCIDParamName          = "aws/defaultVPCID"
	ddinfraDefaultSubnetsParamName        = "aws/defaultSubnets"
	ddinfraDefaultSecurityGroupsParamName = "aws/defaultSecurityGroups"
	ddinfraDefaultInstanceTypeParamName   = "aws/defaultInstanceType"
	ddinfraDefaultKeyPairParamName        = "aws/defaultKeyPairName"

	// AWS ECS
	ddInfraEcsExecKMSKeyID               = "ecs/execKMSKeyID"
	ddInfraEcsTaskExecutionRole          = "ecs/taskExecutionRole"
	ddInfraEcsTaskRole                   = "ecs/taskRole"
	ddInfraEcsServiceAllocatePublicIP    = "ecs/serviceAllocatePublicIP"
	ddInfraEcsFargateCapacityProvider    = "ecs/fargateCapacityProvider"
	ddInfraEcsLinuxECSOptimizedNodeGroup = "ecs/linuxECSOptimizedNodeGroup"
	ddInfraEcsLinuxBottlerocketNodeGroup = "ecs/linuxBottlerocketNodeGroup"
	ddInfraEcsWindowsLTSCNodeGroup       = "ecs/windowsLTSCNodeGroup"
)

type Environment struct {
	config.CommonEnvironment

	awsConfig *sdkconfig.Config
}

func AWSEnvironment(ctx *pulumi.Context) Environment {
	return Environment{
		CommonEnvironment: config.NewCommonEnvironment(ctx),
		awsConfig:         sdkconfig.New(ctx, awsConfigNamespace),
	}
}

// Common
func (e *Environment) Region() string {
	return e.awsConfig.Require(awsRegionParamName)
}

func (e *Environment) DefaultVPCID() string {
	return e.InfraConfig.Require(ddinfraDefaultVPCIDParamName)
}

func (e *Environment) DefaultSubnets() []string {
	var subnets []string
	e.InfraConfig.RequireObject(ddinfraDefaultSubnetsParamName, subnets)
	return subnets
}

func (e *Environment) DefaultSecurityGroups() []string {
	var subnets []string
	e.InfraConfig.RequireObject(ddinfraDefaultSecurityGroupsParamName, subnets)
	return subnets
}

func (e *Environment) DefaultInstanceType() string {
	return e.InfraConfig.Require(ddinfraDefaultInstanceTypeParamName)
}

func (e *Environment) DefaultKeyPairName() string {
	return e.InfraConfig.Require(ddinfraDefaultKeyPairParamName)
}

// ECS
func (e *Environment) ECSExecKMSKeyID() string {
	return e.InfraConfig.Require(ddInfraEcsExecKMSKeyID)
}

func (e *Environment) ECSTaskExecutionRole() string {
	return e.InfraConfig.Require(ddInfraEcsTaskExecutionRole)
}

func (e *Environment) ECSTaskRole() string {
	return e.InfraConfig.Require(ddInfraEcsTaskRole)
}

func (e *Environment) ECSServicePublicIP() bool {
	return e.InfraConfig.RequireBool(ddInfraEcsServiceAllocatePublicIP)
}

func (e *Environment) ECSFargateCapacityProvider() bool {
	return e.InfraConfig.RequireBool(ddInfraEcsFargateCapacityProvider)
}

func (e *Environment) ECSLinuxECSOptimizedNodeGroup() bool {
	return e.InfraConfig.RequireBool(ddInfraEcsLinuxECSOptimizedNodeGroup)
}

func (e *Environment) ECSLinuxBottlerocketNodeGroup() bool {
	return e.InfraConfig.RequireBool(ddInfraEcsLinuxBottlerocketNodeGroup)
}

func (e *Environment) ECSWindowsNodeGroup() bool {
	return e.InfraConfig.RequireBool(ddInfraEcsWindowsLTSCNodeGroup)
}
