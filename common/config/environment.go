package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	namerNamespace         = "common"
	DDInfraConfigNamespace = "ddinfra"
	ddAgentConfigNamespace = "ddagent"

	// Infra namespace
	ddInfraEnvironment       = "env"
	ddInfraKubernetesVersion = "kubernetesVersion"

	// Agent Namespace
	ddAgentDeployParamName        = "deploy"
	ddAgentVersionParamName       = "version"
	ddAgentFullImagePathParamName = "fullImagePath"
	ddAgentAPIKeyParamName        = "apiKey"
	ddAgentAPPKeyParamName        = "appKey"
)

type CommonEnvironment struct {
	Ctx         *pulumi.Context
	InfraConfig *sdkconfig.Config
	AgentConfig *sdkconfig.Config
	CommonNamer namer.Namer
}

func NewCommonEnvironment(ctx *pulumi.Context) CommonEnvironment {
	return CommonEnvironment{
		Ctx:         ctx,
		InfraConfig: sdkconfig.New(ctx, DDInfraConfigNamespace),
		AgentConfig: sdkconfig.New(ctx, ddAgentConfigNamespace),
		CommonNamer: namer.NewNamer(ctx, ""),
	}
}

// Infra namespace
func (e *CommonEnvironment) InfraEnvironmentName() string {
	return e.InfraConfig.Require(ddInfraEnvironment)
}

func (e *CommonEnvironment) KubernetesVersion() string {
	return e.GetStringWithDefault(e.InfraConfig, ddInfraKubernetesVersion, "1.23")
}

func (e *CommonEnvironment) ResourcesTags() pulumi.StringMap {
	defaultTags := pulumi.StringMap{
		"managed-by": pulumi.String("pulumi"),
	}

	// Add user tag
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultTags["username"] = pulumi.String(user.Username)

	// Map environment variables
	lookupVars := []string{"DD_TEAM"}
	for _, varName := range lookupVars {
		if val := os.Getenv(varName); val != "" {
			defaultTags[strings.ToLower(varName)] = pulumi.String(val)
		}
	}

	return defaultTags
}

// Agent Namespace
func (e *CommonEnvironment) AgentDeploy() bool {
	return e.GetBoolWithDefault(e.AgentConfig, ddAgentDeployParamName, false)
}

func (e *CommonEnvironment) AgentVersion() string {
	return e.AgentConfig.Get(ddAgentVersionParamName)
}

func (e *CommonEnvironment) AgentFullImagePath() string {
	return e.AgentConfig.Get(ddAgentFullImagePathParamName)
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

func (e *CommonEnvironment) GetStringListWithDefault(config *sdkconfig.Config, paramName string, defaultValue []string) []string {
	val, err := config.Try(paramName)
	if err == nil {
		return strings.Split(val, ",")
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

func (e *CommonEnvironment) GetIntWithDefault(config *sdkconfig.Config, paramName string, defaultValue int) int {
	val, err := config.TryInt(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue

}

type Environment interface {
	DefaultInstanceType() string
	DefaultARMInstanceType() string
	GetCommonEnvironment() *CommonEnvironment
	DefaultPrivateKeyPath() string
	DefaultPrivateKeyPassword() string
}
