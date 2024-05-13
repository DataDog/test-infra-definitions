package azure

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"

	sdkazure "github.com/pulumi/pulumi-azure-native-sdk/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	azConfigNamespace = "azure-native"
	azNamerNamespace  = "az"

	// Azure Infra
	DDInfraDefaultResourceGroup            = "az/defaultResourceGroup"
	DDInfraDefaultVNetParamName            = "az/defaultVNet"
	DDInfraDefaultSubnetParamName          = "az/defaultSubnet"
	DDInfraDefaultSecurityGroupParamName   = "az/defaultSecurityGroup"
	DDInfraDefaultInstanceTypeParamName    = "az/defaultInstanceType"
	DDInfraDefaultARMInstanceTypeParamName = "az/defaultARMInstanceType"
	DDInfraDefaultPublicKeyPath            = "az/defaultPublicKeyPath"
	DDInfraDefaultPrivateKeyPath           = "az/defaultPrivateKeyPath"
	DDInfraDefaultPrivateKeyPassword       = "az/defaultPrivateKeyPassword"
	DDInfraAksLinuxKataNodeGroup           = "az/aks/linuxKataNodeGroup"
)

type Environment struct {
	*config.CommonEnvironment

	Namer namer.Namer

	envDefault environmentDefault
}

var _ config.Env = (*Environment)(nil)

func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	env := Environment{
		Namer: namer.NewNamer(ctx, azNamerNamespace),
	}
	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}
	env.CommonEnvironment = &commonEnv
	env.envDefault = getEnvironmentDefault(config.FindEnvironmentName(commonEnv.InfraEnvironmentNames(), azNamerNamespace))

	azureProvider, err := sdkazure.NewProvider(ctx, string(config.ProviderAzure), &sdkazure.ProviderArgs{
		DisablePulumiPartnerId: pulumi.BoolPtr(true),
		SubscriptionId:         pulumi.StringPtr(env.envDefault.azure.subscriptionID),
		TenantId:               pulumi.StringPtr(env.envDefault.azure.tenantID),
	})
	if err != nil {
		return Environment{}, err
	}
	env.RegisterProvider(config.ProviderAzure, azureProvider)

	return env, nil
}

// Cross Cloud Provider config
func (e *Environment) InternalRegistry() string {
	return "none"
}

func (e *Environment) InternalDockerhubMirror() string {
	return "registry-1.docker.io"
}

// Common
func (e *Environment) DefaultResourceGroup() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultResourceGroup, e.envDefault.ddInfra.defaultResourceGroup)
}

func (e *Environment) DefaultVNet() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultVNetParamName, e.envDefault.ddInfra.defaultVNet)
}

func (e *Environment) DefaultSubnet() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultSubnetParamName, e.envDefault.ddInfra.defaultSubnet)
}

func (e *Environment) DefaultSecurityGroup() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultSecurityGroupParamName, e.envDefault.ddInfra.defaultSecurityGroup)
}

func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultARMInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultARMInstanceTypeParamName, e.envDefault.ddInfra.defaultARMInstanceType)
}

func (e *Environment) DefaultPublicKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPublicKeyPath)
}

func (e *Environment) DefaultPrivateKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPath)
}

func (e *Environment) DefaultPrivateKeyPassword() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPassword)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}

// LinuxKataNodeGroup Whether to deploy a kata node pool
func (e *Environment) LinuxKataNodeGroup() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraAksLinuxKataNodeGroup, e.envDefault.ddInfra.aks.linuxKataNodeGroup)
}
