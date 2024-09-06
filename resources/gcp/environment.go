package gcp

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	gcpConfigNamespace = "gcp"
	gcpNamerNamespace  = "gcp"

	// GCP Infra
	DDInfraDefaultPublicKeyPath            = "gcp/defaultPublicKeyPath"
	DDInfraDefaultPrivateKeyPath           = "gcp/defaultPrivateKeyPath"
	DDInfraDefaultPrivateKeyPassword       = "gcp/defaultPrivateKeyPassword"
	DDInfraDefaultInstanceTypeParamName    = "gcp/defaultInstanceType"
	DDInfraDefaultNetworkNameParamName     = "gcp/defaultNetworkName"
	DDInfraDefaultSubnetNameParamName      = "gcp/defaultSubnet"
	DDInfraDefautVMServiceAccountParamName = "gcp/defaultVMServiceAccount"
)

type Environment struct {
	*config.CommonEnvironment

	Namer namer.Namer

	envDefault environmentDefault
}

var _ config.Env = (*Environment)(nil)

func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	env := Environment{
		Namer: namer.NewNamer(ctx, gcpNamerNamespace),
	}
	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}
	env.CommonEnvironment = &commonEnv
	env.envDefault = getEnvironmentDefault(config.FindEnvironmentName(commonEnv.InfraEnvironmentNames(), gcpNamerNamespace))

	gcpProvider, err := gcp.NewProvider(ctx, string(config.ProviderGCP), &gcp.ProviderArgs{
		Project: pulumi.StringPtr(env.envDefault.gcp.project),
		Zone:    pulumi.StringPtr(env.envDefault.gcp.region),
	})
	if err != nil {
		return Environment{}, err
	}
	env.RegisterProvider(config.ProviderGCP, gcpProvider)

	return env, nil
}

// Cross Cloud Provider config

func (e *Environment) InternalRegistry() string {
	return "none"
}

func (e *Environment) InternalDockerhubMirror() string {
	return "registry-1.docker.io"
}

func (e *Environment) InternalRegistryImageTagExists(_, _ string) (bool, error) {
	return true, nil
}

// Common

func (e *Environment) DefaultPublicKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPublicKeyPath)
}

func (e *Environment) DefaultPrivateKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPath)
}

func (e *Environment) DefaultPrivateKeyPassword() string {
	return e.InfraConfig.Get(DDInfraDefaultPrivateKeyPassword)
}

func (e *Environment) DefaultNetworkName() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultNetworkNameParamName, e.envDefault.ddInfra.defaultNetworkName)
}

func (e *Environment) DefaultSubnet() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultSubnetNameParamName, e.envDefault.ddInfra.defaultSubnetName)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}
func (e *Environment) DefaultInstanceType() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefaultInstanceTypeParamName, e.envDefault.ddInfra.defaultInstanceType)
}

func (e *Environment) DefaultVMServiceAccount() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraDefautVMServiceAccountParamName, e.envDefault.ddInfra.defaultVMServiceAccount)
}
