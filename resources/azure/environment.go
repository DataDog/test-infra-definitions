package azure

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"

	sdkazure "github.com/pulumi/pulumi-azure-native-sdk"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	azConfigNamespace = "azure-native"
	azNamerNamespace  = "az"

	// Azure Infra
	ddInfraDefaultResourceGroup            = "az/defaultResourceGroup"
	ddinfraDefaultVNetParamName            = "az/defaultVNet"
	ddinfraDefaultSubnetParamName          = "az/defaultSubnet"
	ddinfraDefaultSecurityGroupParamName   = "az/defaultSecurityGroup"
	ddinfraDefaultInstanceTypeParamName    = "az/defaultInstanceType"
	ddinfraDefaultARMInstanceTypeParamName = "az/defaultARMInstanceType"
	ddinfraDefaultPublicKeyPath            = "az/defaultPublicKeyPath"
	ddinfraDefaultPrivateKeyPath           = "az/defaultPrivateKeyPath"
	ddinfraDefaultPrivateKeyPassword       = "az/defaultPrivateKeyPassword"
)

type Environment struct {
	*config.CommonEnvironment

	Namer namer.Namer

	envDefault environmentDefault
}

func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}

	env := Environment{
		CommonEnvironment: &commonEnv,
		Namer:             namer.NewNamer(ctx, azNamerNamespace),
		envDefault:        getEnvironmentDefault(config.FindEnvironmentName(commonEnv.InfraEnvironmentNames(), azNamerNamespace)),
	}

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

// Common
func (e *Environment) DefaultResourceGroup() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraDefaultResourceGroup, e.envDefault.ddInfra.defaultResourceGroup)
}

func (e *Environment) DefaultVNet() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultVNetParamName, e.envDefault.ddInfra.defaultVNet)
}

func (e *Environment) DefaultSubnet() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultSubnetParamName, e.envDefault.ddInfra.defaultSubnet)
}

func (e *Environment) DefaultSecurityGroup() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultSecurityGroupParamName, e.envDefault.ddInfra.defaultSecurityGroup)
}

func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultARMInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, ddinfraDefaultARMInstanceTypeParamName, e.envDefault.ddInfra.defaultARMInstanceType)
}

func (e *Environment) DefaultPublicKeyPath() string {
	return e.InfraConfig.Get(ddinfraDefaultPublicKeyPath)
}

func (e *Environment) DefaultPrivateKeyPath() string {
	return e.InfraConfig.Get(ddinfraDefaultPrivateKeyPath)
}

func (e *Environment) DefaultPrivateKeyPassword() string {
	return e.InfraConfig.Get(ddinfraDefaultPrivateKeyPassword)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}
