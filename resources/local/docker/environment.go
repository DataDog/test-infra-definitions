package docker

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Environment struct {
	*config.CommonEnvironment

	Namer         namer.Namer
	dockerNetwork *docker.Network
}

var _ config.Env = (*Environment)(nil)

func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	env := Environment{
		Namer: namer.NewNamer(ctx, "dclocal"),
	}

	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}

	env.CommonEnvironment = &commonEnv

	dockerProvider, err := docker.NewProvider(ctx, string(config.ProviderDocker), nil)
	if err != nil {
		return Environment{}, err
	}
	env.RegisterProvider(config.ProviderDocker, dockerProvider)

	// Create a Docker network
	dn, err := docker.NewNetwork(ctx, "network", &docker.NetworkArgs{
		Name: pulumi.String(fmt.Sprintf("local-e2e-%v", ctx.Stack())),
	}, env.WithProviders(config.ProviderDocker))
	if err != nil {
		return Environment{}, err
	}

	env.dockerNetwork = dn

	return env, nil
}

func (e *Environment) InternalDockerhubMirror() string {
	return "bob"
}

func (e *Environment) InternalRegistry() string {
	return "bob"
}
