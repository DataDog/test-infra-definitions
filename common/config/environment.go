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

	// Infra namespace
	ddInfraEnvironment       = "env"
	ddInfraKubernetesVersion = "kubernetesVersion"

	// Agent Namespace
	ddAgentDeployParamName  = "deploy"
	ddAgentVersionParamName = "version"
	ddAgentAPIKeyParamName  = "apiKey"
	ddAgentAPPKeyParamName  = "appKey"
)

type CommonEnvironment struct {
	Ctx         *pulumi.Context
	InfraConfig *sdkconfig.Config
	AgentConfig *sdkconfig.Config
}

func NewCommonEnvironment(ctx *pulumi.Context) CommonEnvironment {
	return CommonEnvironment{
		Ctx:         ctx,
		InfraConfig: sdkconfig.New(ctx, ddInfraConfigNamespace),
		AgentConfig: sdkconfig.New(ctx, ddAgentConfigNamespace),
	}
}

// Infra namespace
func (e *CommonEnvironment) InfraEnvironmentName() string {
	return e.InfraConfig.Require(ddInfraEnvironment)
}

func (e *CommonEnvironment) KubernetesVersion() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraKubernetesVersion, "1.23")
}

// Agent Namespace
func (e *CommonEnvironment) AgentDeploy() bool {
	return e.GetBoolWithDefault(e.AgentConfig, ddAgentDeployParamName, true)
}

func (e *CommonEnvironment) AgentVersion() string {
	return e.AgentConfig.Get(ddAgentVersionParamName)
}

func (e *CommonEnvironment) AgentAPIKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(ddAgentAPIKeyParamName)
}

func (e *CommonEnvironment) AgentAPPKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(ddAgentAPPKeyParamName)
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

func (e *CommonEnvironment) GetStringWithDefault(config *sdkconfig.Config, paramName string, defaultValue string) string {
	val, err := config.Try(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}

func (e *CommonEnvironment) GetObjectWithDefault(config *sdkconfig.Config, paramName string, outputValue, defaultValue interface{}) interface{} {
	err := config.TryObject(paramName, outputValue)
	if err == nil {
		return outputValue
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}
