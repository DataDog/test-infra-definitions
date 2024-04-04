package aws

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"

	sdkaws "github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	awsConfigNamespace = "aws"
	awsRegionParamName = "region"

	// AWS Infra
	DDInfraDefaultVPCIDParamName           = "aws/defaultVPCID"
	DDInfraDefaultSubnetsParamName         = "aws/defaultSubnets"
	DDInfraDefaultSecurityGroupsParamName  = "aws/defaultSecurityGroups"
	DDInfraDefaultInstanceTypeParamName    = "aws/defaultInstanceType"
	DDInfraDefaultInstanceProfileParamName = "aws/defaultInstanceProfile"
	DDInfraDefaultARMInstanceTypeParamName = "aws/defaultARMInstanceType"
	DDInfraDefaultKeyPairParamName         = "aws/defaultKeyPairName"
	DDinfraDefaultPublicKeyPath            = "aws/defaultPublicKeyPath"
	DDInfraDefaultPrivateKeyPath           = "aws/defaultPrivateKeyPath"
	DDInfraDefaultPrivateKeyPassword       = "aws/defaultPrivateKeyPassword"
	DDInfraDefaultInstanceStorageSize      = "aws/defaultInstanceStorageSize"
	DDInfraDefaultShutdownBehavior         = "aws/defaultShutdownBehavior"
	DDInfraDefaultInternalRegistry         = "aws/defaultInternalRegistry"

	// AWS ECS
	DDInfraEcsExecKMSKeyID                  = "aws/ecs/execKMSKeyID"
	DDInfraEcsFargateFakeintakeClusterArn   = "aws/ecs/fargateFakeintakeClusterArn"
	DDInfraEcsFakeintakeLBListenerArn       = "aws/ecs/fakeintakeLBListenerArn"
	DDInfraEcsFakeintakeLBBaseHost          = "aws/ecs/fakeintakeLBBaseHost"
	DDInfraEcsTaskExecutionRole             = "aws/ecs/taskExecutionRole"
	DDInfraEcsTaskRole                      = "aws/ecs/taskRole"
	DDInfraEcsInstanceProfile               = "aws/ecs/instanceProfile"
	DDInfraEcsServiceAllocatePublicIP       = "aws/ecs/serviceAllocatePublicIP"
	DDInfraEcsFargateCapacityProvider       = "aws/ecs/fargateCapacityProvider"
	DDInfraEcsLinuxECSOptimizedNodeGroup    = "aws/ecs/linuxECSOptimizedNodeGroup"
	DDInfraEcsLinuxECSOptimizedARMNodeGroup = "aws/ecs/linuxECSOptimizedARMNodeGroup"
	DDInfraEcsLinuxBottlerocketNodeGroup    = "aws/ecs/linuxBottlerocketNodeGroup"
	DDInfraEcsWindowsLTSCNodeGroup          = "aws/ecs/windowsLTSCNodeGroup"

	// AWS EKS
	DDInfraEksAllowedInboundSecurityGroups = "aws/eks/inboundSecurityGroups"
	DDInfraEksAllowedInboundPrefixList     = "aws/eks/inboundPrefixLists"
	DDInfraEksFargateNamespace             = "aws/eks/fargateNamespace"
	DDInfraEksLinuxNodeGroup               = "aws/eks/linuxNodeGroup"
	DDInfraEksLinuxARMNodeGroup            = "aws/eks/linuxARMNodeGroup"
	DDInfraEksLinuxBottlerocketNodeGroup   = "aws/eks/linuxBottlerocketNodeGroup"
	DDInfraEksWindowsNodeGroup             = "aws/eks/windowsNodeGroup"
)

type Environment struct {
	*config.CommonEnvironment

	Namer namer.Namer

	awsConfig  *sdkconfig.Config
	envDefault environmentDefault

	randomSubnets pulumi.StringArrayOutput
}

var _ config.CloudProviderEnvironment = (*Environment)(nil)

func WithCommonEnvironment(e *config.CommonEnvironment) func(*Environment) {
	return func(awsEnv *Environment) {
		awsEnv.CommonEnvironment = e
		awsEnv.CommonEnvironment.CloudProviderEnvironment = awsEnv
	}
}

func NewEnvironment(ctx *pulumi.Context, options ...func(*Environment)) (Environment, error) {
	env := Environment{
		Namer:     namer.NewNamer(ctx, awsConfigNamespace),
		awsConfig: sdkconfig.New(ctx, awsConfigNamespace),
	}

	for _, opt := range options {
		opt(&env)
	}

	if env.CommonEnvironment == nil {
		commonEnv, err := config.NewCommonEnvironment(ctx, &env)
		if err != nil {
			return Environment{}, err
		}

		env.CommonEnvironment = &commonEnv
	}
	env.envDefault = getEnvironmentDefault(config.FindEnvironmentName(env.CommonEnvironment.InfraEnvironmentNames(), awsConfigNamespace))

	awsProvider, err := sdkaws.NewProvider(ctx, string(config.ProviderAWS), &sdkaws.ProviderArgs{
		Region: pulumi.String(env.Region()),
		DefaultTags: sdkaws.ProviderDefaultTagsArgs{
			Tags: env.CommonEnvironment.ResourcesTags(),
		},
		SkipCredentialsValidation: pulumi.BoolPtr(false),
		SkipMetadataApiCheck:      pulumi.BoolPtr(false),
	})
	if err != nil {
		return Environment{}, err
	}
	env.RegisterProvider(config.ProviderAWS, awsProvider)

	shuffle, err := random.NewRandomShuffle(env.Ctx, env.Namer.ResourceName("rnd-subnet"), &random.RandomShuffleArgs{
		Inputs:      pulumi.ToStringArray(env.DefaultSubnets()),
		ResultCount: pulumi.IntPtr(2),
	}, env.WithProviders(config.ProviderRandom))
	if err != nil {
		return Environment{}, err
	}
	env.randomSubnets = shuffle.Results

	return env, nil
}

// Cross Cloud Provider config
func (e *Environment) InternalRegistry() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInternalRegistry, e.envDefault.ddInfra.defaultInternalRegistry)
}

// Common
func (e *Environment) Region() string {
	return e.GetStringWithDefault(e.awsConfig, awsRegionParamName, e.envDefault.aws.region)
}

func (e *Environment) DefaultVPCID() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultVPCIDParamName, e.envDefault.ddInfra.defaultVPCID)
}

func (e *Environment) DefaultSubnets() []string {
	return e.GetStringListWithDefault(e.InfraConfig, DDInfraDefaultSubnetsParamName, e.envDefault.ddInfra.defaultSubnets)
}

func (e *Environment) RandomSubnets() pulumi.StringArrayOutput {
	return e.randomSubnets
}

func (e *Environment) DefaultSecurityGroups() []string {
	return e.GetStringListWithDefault(e.InfraConfig, DDInfraDefaultSecurityGroupsParamName, e.envDefault.ddInfra.defaultSecurityGroups)
}

func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultInstanceProfileName() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInstanceProfileParamName, e.envDefault.ddInfra.defaultInstanceProfileName)
}

func (e *Environment) DefaultARMInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultARMInstanceTypeParamName, e.envDefault.ddInfra.defaultARMInstanceType)
}

func (e *Environment) DefaultKeyPairName() string {
	// No default value for keyPair
	return e.InfraConfig.Require(DDInfraDefaultKeyPairParamName)
}

func (e *Environment) DefaultPublicKeyPath() string {
	return e.InfraConfig.Require(DDinfraDefaultPublicKeyPath)
}

func (e *Environment) DefaultPrivateKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPath)
}

func (e *Environment) DefaultPrivateKeyPassword() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPassword)
}

func (e *Environment) DefaultInstanceStorageSize() int {
	return e.GetIntWithDefault(e.InfraConfig, DDInfraDefaultInstanceStorageSize, e.envDefault.ddInfra.defaultInstanceStorageSize)
}

// shutdown behavior can be 'terminate' or 'stop'
func (e *Environment) DefaultShutdownBehavior() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultShutdownBehavior, e.envDefault.ddInfra.defaultShutdownBehavior)
}

// ECS
func (e *Environment) ECSExecKMSKeyID() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsExecKMSKeyID, e.envDefault.ddInfra.ecs.execKMSKeyID)
}

func (e *Environment) ECSFargateFakeintakeClusterArn() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsFargateFakeintakeClusterArn, e.envDefault.ddInfra.ecs.fargateFakeintakeClusterArn)
}

func (e *Environment) ECSFakeintakeLBListenerArn() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsFakeintakeLBListenerArn, e.envDefault.ddInfra.ecs.fakeintakeLBListenerArn)
}

func (e *Environment) ECSFakeintakeLBBaseHost() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsFakeintakeLBBaseHost, e.envDefault.ddInfra.ecs.fakeintakeLBBaseHostHeader)
}

func (e *Environment) ECSTaskExecutionRole() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsTaskExecutionRole, e.envDefault.ddInfra.ecs.taskExecutionRole)
}

func (e *Environment) ECSTaskRole() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsTaskRole, e.envDefault.ddInfra.ecs.taskRole)
}

func (e *Environment) ECSInstanceProfile() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEcsInstanceProfile, e.envDefault.ddInfra.ecs.instanceProfile)
}

func (e *Environment) ECSServicePublicIP() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsServiceAllocatePublicIP, e.envDefault.ddInfra.ecs.serviceAllocatePublicIP)
}

func (e *Environment) ECSFargateCapacityProvider() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsFargateCapacityProvider, e.envDefault.ddInfra.ecs.fargateCapacityProvider)
}

func (e *Environment) ECSLinuxECSOptimizedNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsLinuxECSOptimizedNodeGroup, e.envDefault.ddInfra.ecs.linuxECSOptimizedNodeGroup)
}

func (e *Environment) ECSLinuxECSOptimizedARMNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsLinuxECSOptimizedARMNodeGroup, e.envDefault.ddInfra.ecs.linuxECSOptimizedARMNodeGroup)
}

func (e *Environment) ECSLinuxBottlerocketNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsLinuxBottlerocketNodeGroup, e.envDefault.ddInfra.ecs.linuxBottlerocketNodeGroup)
}

func (e *Environment) ECSWindowsNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEcsWindowsLTSCNodeGroup, e.envDefault.ddInfra.ecs.windowsLTSCNodeGroup)
}

func (e *Environment) EKSAllowedInboundSecurityGroups() []string {
	var arr []string
	resObj := e.GetObjectWithDefault(e.InfraConfig, DDInfraEksAllowedInboundSecurityGroups, arr, e.envDefault.ddInfra.eks.allowedInboundSecurityGroups)
	return resObj.([]string)
}

func (e *Environment) EKSAllowedInboundPrefixLists() []string {
	var arr []string
	resObj := e.GetObjectWithDefault(e.InfraConfig, DDInfraEksAllowedInboundPrefixList, arr, e.envDefault.ddInfra.eks.allowedInboundPrefixList)
	return resObj.([]string)
}

func (e *Environment) EKSFargateNamespace() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraEksFargateNamespace, e.envDefault.ddInfra.eks.fargateNamespace)
}

func (e *Environment) EKSLinuxNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEksLinuxNodeGroup, e.envDefault.ddInfra.eks.linuxNodeGroup)
}

func (e *Environment) EKSLinuxARMNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEksLinuxARMNodeGroup, e.envDefault.ddInfra.eks.linuxARMNodeGroup)
}

func (e *Environment) EKSBottlerocketNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEksLinuxBottlerocketNodeGroup, e.envDefault.ddInfra.eks.linuxBottlerocketNodeGroup)
}

func (e *Environment) EKSWindowsNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraEksWindowsNodeGroup, e.envDefault.ddInfra.eks.windowsLTSCNodeGroup)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}
