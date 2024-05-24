package local

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Environment struct {
	// CommonEnvironment is the common environment for all the components
	// in the test-infra-definitions.
	*config.CommonEnvironment
}

var _ config.Env = (*Environment)(nil)

// NewEnvironment creates a new local environment.
func NewEnvironment(ctx *pulumi.Context) (Environment, error) {
	env := Environment{}

	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return Environment{}, err
	}

	env.CommonEnvironment = &commonEnv

	return env, nil
}

// InternalRegistry returns the internal registry.
func (e *Environment) InternalRegistry() string {
	return "none"
}

// InternalDockerhubMirror returns the internal Dockerhub mirror.
func (e *Environment) InternalDockerhubMirror() string {
	return "registry-1.docker.io"
}
