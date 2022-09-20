package config

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	ddInfraConfigNamespace = "ddinfra"
	ddAgentConfigNamespace = "ddagent"

	// Agent Namespace
	ddAgentDeployParamName = "deploy"
	ddAgentAPIKeyParamName = "apiKey"
)

type CommonEnvironment struct {
	Ctx         *pulumi.Context
	InfraConfig *sdkconfig.Config
	AgentConfig *sdkconfig.Config
}

func NewCommonEnvironment(ctx *pulumi.Context) CommonEnvironment {
	return CommonEnvironment{
		InfraConfig: sdkconfig.New(ctx, ddInfraConfigNamespace),
		AgentConfig: sdkconfig.New(ctx, ddAgentConfigNamespace),
	}
}

func (e *CommonEnvironment) DeployAgent() bool {
	return e.GetBoolWithDefault(e.AgentConfig, ddAgentDeployParamName, true)
}

func (e *CommonEnvironment) AgentAPIKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(ddAgentAPIKeyParamName)
}

func (e *CommonEnvironment) GetBoolWithDefault(config *sdkconfig.Config, paramName string, defaultValue bool) bool {
	val, err := config.TryBool(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}
