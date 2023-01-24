package aws

import (
	"fmt"

	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"

	sdkaws "github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	awsConfigNamespace = "aws"
	awsRegionParamName = "region"

	// AWS Infra
	ddinfraDefaultVPCIDParamName           = "aws/defaultVPCID"
	ddinfraDefaultSubnetsParamName         = "aws/defaultSubnets"
	ddinfraDefaultSecurityGroupsParamName  = "aws/defaultSecurityGroups"
	DDInfraDefaultInstanceTypeParamName    = "aws/defaultInstanceType"
	DDInfraDefaultARMInstanceTypeParamName = "aws/defaultARMInstanceType"
	DDInfraDefaultKeyPairParamName         = "aws/defaultKeyPairName"
	ddinfraDefaultPrivateKeyPath           = "aws/defaultPrivateKeyPath"
	ddinfraDefaultPrivateKeyPassword       = "aws/defaultPrivateKeyPassword"

	// AWS ECS
	ddInfraEcsExecKMSKeyID                  = "aws/ecs/execKMSKeyID"
	ddInfraEcsTaskExecutionRole             = "aws/ecs/taskExecutionRole"
	ddInfraEcsTaskRole                      = "aws/ecs/taskRole"
	ddInfraEcsInstanceProfile               = "aws/ecs/instanceProfile"
	ddInfraEcsServiceAllocatePublicIP       = "aws/ecs/serviceAllocatePublicIP"
	ddInfraEcsFargateCapacityProvider       = "aws/ecs/fargateCapacityProvider"
	ddInfraEcsLinuxECSOptimizedNodeGroup    = "aws/ecs/linuxECSOptimizedNodeGroup"
	ddInfraEcsLinuxECSOptimizedARMNodeGroup = "aws/ecs/linuxECSOptimizedARMNodeGroup"
	ddInfraEcsLinuxBottlerocketNodeGroup    = "aws/ecs/linuxBottlerocketNodeGroup"
	ddInfraEcsWindowsLTSCNodeGroup          = "aws/ecs/windowsLTSCNodeGroup"

	// AWS EKS
	ddInfraEksAllowedInboundSecurityGroups = "aws/eks/clusterSecurityGroups"
	ddInfraEksFargateNamespace             = "aws/eks/fargateNamespace"
	ddInfraEksLinuxNodeGroup               = "aws/eks/linuxNodeGroup"
	ddInfraEksLinuxARMNodeGroup            = "aws/eks/linuxARMNodeGroup"
	ddInfraEksLinuxBottlerocketNodeGroup   = "aws/eks/linuxBottlerocketNodeGroup"
	ddInfraEksWindowsNodeGroup             = "aws/eks/windowsNodeGroup"
)

type Environment struct {
	*config.CommonEnvironment

	Provider *sdkaws.Provider
	Namer    namer.Namer

	awsConfig  *sdkconfig.Config
	envDefault environmentDefault
}

func AWSEnvironment(ctx *pulumi.Context) (Environment, error) {
	commonEnv := config.NewCommonEnvironment(ctx)

	env := Environment{
		CommonEnvironment: &commonEnv,
		Namer:             namer.NewNamer(ctx, awsConfigNamespace),
		awsConfig:         sdkconfig.New(ctx, awsConfigNamespace),
		envDefault:        getEnvironmentDefault(commonEnv.InfraEnvironmentName()),
	}

	var err error
	env.Provider, err = sdkaws.NewProvider(ctx, "aws", &sdkaws.ProviderArgs{
		Region: pulumi.String(env.Region()),
		DefaultTags: sdkaws.ProviderDefaultTagsArgs{
			Tags: commonEnv.ResourcesTags(),
		},
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
	return e.GetStringListWithDefault(e.InfraConfig, ddinfraDefaultSubnetsParamName, e.envDefault.ddInfra.defaultSubnets)
}

func (e *Environment) DefaultSecurityGroups() []string {
	return e.GetStringListWithDefault(e.InfraConfig, ddinfraDefaultSecurityGroupsParamName, e.envDefault.ddInfra.defaultSecurityGroups)
}

func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultARMInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultARMInstanceTypeParamName, e.envDefault.ddInfra.defaultARMInstanceType)
}

func (e *Environment) DefaultKeyPairName() string {
	// No default value for keyPair
	return e.InfraConfig.Require(DDInfraDefaultKeyPairParamName)
}

func (e *Environment) DefaultPrivateKeyPath() string {
	return e.InfraConfig.Get(ddinfraDefaultPrivateKeyPath)
}

func (e *Environment) DefaultPrivateKeyPassword() string {
	return e.InfraConfig.Get(ddinfraDefaultPrivateKeyPassword)
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

func (e *Environment) ECSLinuxECSOptimizedARMNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEcsLinuxECSOptimizedARMNodeGroup, e.envDefault.ddInfra.ecs.linuxECSOptimizedARMNodeGroup)
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

func (e *Environment) EKSFargateNamespace() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraEksFargateNamespace, e.envDefault.ddInfra.eks.fargateNamespace)
}

func (e *Environment) EKSLinuxNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEksLinuxNodeGroup, e.envDefault.ddInfra.eks.linuxNodeGroup)
}

func (e *Environment) EKSLinuxARMNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEksLinuxARMNodeGroup, e.envDefault.ddInfra.eks.linuxARMNodeGroup)
}

func (e *Environment) EKSBottlerocketNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEksLinuxBottlerocketNodeGroup, e.envDefault.ddInfra.eks.linuxBottlerocketNodeGroup)
}

func (e *Environment) EKSWindowsNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, ddInfraEksWindowsNodeGroup, e.envDefault.ddInfra.eks.windowsLTSCNodeGroup)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}

func GetInfraKey(keyName string) string {
	return fmt.Sprintf("%v:%v", config.DDInfraConfigNamespace, keyName)
}
