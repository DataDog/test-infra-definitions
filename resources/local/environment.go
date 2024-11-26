package local

import (
	config "github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	localNamerNamespace = "local"
	// local Infra (local)
	DDInfraDefaultPublicKeyPath = "local/defaultPublicKeyPath"
)

type Environment struct {
	*config.CommonEnvironment

	Namer namer.Namer
}

var _ config.Env = (*Environment)(nil)

func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	env := Environment{
		Namer: namer.NewNamer(ctx, localNamerNamespace),
	}

	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}

	env.CommonEnvironment = &commonEnv

	return env, nil
}

// Cross Cloud Provider config

// InternalRegistry returns the internal registry.
func (e *Environment) InternalRegistry() string {
	return "none"
}

// InternalDockerhubMirror returns the internal Dockerhub mirror.
func (e *Environment) InternalDockerhubMirror() string {
	return "registry-1.docker.io"
}

// InternalRegistryImageTagExists returns true if the image tag exists in the internal registry.
func (e *Environment) InternalRegistryImageTagExists(_, _ string) (bool, error) {
	return true, nil
}

// Common
func (e *Environment) DefaultPublicKeyPath() string {
	return e.InfraConfig.Get(DDInfraDefaultPublicKeyPath)
}

func (e *Environment) GetCommonEnvironment() *config.CommonEnvironment {
	return e.CommonEnvironment
}
