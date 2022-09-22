package aws

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	sdkaws "github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
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
	ddInfraEcsExecKMSKeyID               = "aws/ecs/execKMSKeyID"
	ddInfraEcsTaskExecutionRole          = "aws/ecs/taskExecutionRole"
	ddInfraEcsTaskRole                   = "aws/ecs/taskRole"
	ddInfraEcsInstanceProfile            = "aws/ecs/instanceProfile"
	ddInfraEcsServiceAllocatePublicIP    = "aws/ecs/serviceAllocatePublicIP"
	ddInfraEcsFargateCapacityProvider    = "aws/ecs/fargateCapacityProvider"
	ddInfraEcsLinuxECSOptimizedNodeGroup = "aws/ecs/linuxECSOptimizedNodeGroup"
	ddInfraEcsLinuxBottlerocketNodeGroup = "aws/ecs/linuxBottlerocketNodeGroup"
	ddInfraEcsWindowsLTSCNodeGroup       = "aws/ecs/windowsLTSCNodeGroup"

	// AWS EKS
	ddInfraEksAllowedInboundSecurityGroups = "aws/eks/clusterSecurityGroups"
)

type Environment struct {
	*config.CommonEnvironment

	Provider *sdkaws.Provider

	awsConfig  *sdkconfig.Config
	envDefault environmentDefault
}

func AWSEnvironment(ctx *pulumi.Context) (Environment, error) {
	commonEnv := config.NewCommonEnvironment(ctx)

	env := Environment{
		CommonEnvironment: &commonEnv,
		awsConfig:         sdkconfig.New(ctx, awsConfigNamespace),
		envDefault:        getEnvironmentDefault(commonEnv.InfraEnvironmentName()),
	}

	var err error
	env.Provider, err = sdkaws.NewProvider(ctx, "aws", &sdkaws.ProviderArgs{
		Region:                 pulumi.String(env.Region()),
		SharedCredentialsFiles: pulumi.StringArray{pulumi.String("~/.aws/credentials")},
	})

	if err != nil {
		return Environment{}, err
	}

	return env, nil
}

// Common
func (e *Environment) Region() string {
	return e.GetStringWithDefault(e.awsConfig, awsRegionParamName, e.envDefault.aws.region)
}

func (e *Environment) DefaultVPCID() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultVPCIDParamName, e.envDefault.ddInfra.defaultVPCID)
}

func (e *Environment) DefaultSubnets() []string {
	var arr []string
	resInt := e.GetObjectWithDefault(e.InfraConfig, ddinfraDefaultSubnetsParamName, arr, e.envDefault.ddInfra.defaultSubnets)
	return resInt.([]string)
}

func (e *Environment) DefaultSecurityGroups() []string {
	var arr []string
	resInt := e.GetObjectWithDefault(e.InfraConfig, ddinfraDefaultSecurityGroupsParamName, arr, e.envDefault.ddInfra.defaultSecurityGroups)
	return resInt.([]string)
}

func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultKeyPairName() string {
	// No default value for keyPair
	return e.InfraConfig.Require(ddinfraDefaultKeyPairParamName)
}

// ECS
func (e *Environment) ECSExecKMSKeyID() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraEcsExecKMSKeyID, e.envDefault.ddInfra.ecs.execKMSKeyID)
}

func (e *Environment) ECSTaskExecutionRole() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraEcsTaskExecutionRole, e.envDefault.ddInfra.ecs.taskExecutionRole)
}

func (e *Environment) ECSTaskRole() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraEcsTaskRole, e.envDefault.ddInfra.ecs.taskRole)
}

func (e *Environment) ECSInstanceProfile() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraEcsInstanceProfile, e.envDefault.ddInfra.ecs.instanceProfile)
}

func (e *Environment) ECSServicePublicIP() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsServiceAllocatePublicIP, e.envDefault.ddInfra.ecs.serviceAllocatePublicIP)
}

func (e *Environment) ECSFargateCapacityProvider() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsFargateCapacityProvider, e.envDefault.ddInfra.ecs.fargateCapacityProvider)
}

func (e *Environment) ECSLinuxECSOptimizedNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsLinuxECSOptimizedNodeGroup, e.envDefault.ddInfra.ecs.linuxECSOptimizedNodeGroup)
}

func (e *Environment) ECSLinuxBottlerocketNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsLinuxBottlerocketNodeGroup, e.envDefault.ddInfra.ecs.linuxBottlerocketNodeGroup)
}

func (e *Environment) ECSWindowsNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsWindowsLTSCNodeGroup, e.envDefault.ddInfra.ecs.windowsLTSCNodeGroup)
}

func (e *Environment) EKSAllowedInboundSecurityGroups() []string {
	var arr []string
	resInt := e.GetObjectWithDefault(e.InfraConfig, ddInfraEksAllowedInboundSecurityGroups, arr, e.envDefault.ddInfra.eks.allowedInboundSecurityGroups)
	return resInt.([]string)
}
