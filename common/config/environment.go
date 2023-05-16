package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-command/sdk/go/command"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	multiValueSeparator = ","

	namerNamespace         = "common"
	DDInfraConfigNamespace = "ddinfra"
	DDAgentConfigNamespace = "ddagent"

	// Infra namespace
	DDInfraEnvironment       = "env"
	DDInfraKubernetesVersion = "kubernetesVersion"
	DDInfraOSFamily          = "osFamily"

	// Agent Namespace
	DDAgentDeployParamName        = "deploy"
	DDAgentVersionParamName       = "version"
	DDAgentFullImagePathParamName = "fullImagePath"
	DDAgentAPIKeyParamName        = "apiKey"
	DDAgentAPPKeyParamName        = "appKey"
)

type CommonEnvironment struct {
	Ctx         *pulumi.Context
	InfraConfig *sdkconfig.Config
	AgentConfig *sdkconfig.Config
	CommonNamer namer.Namer

	RandomProvider  *random.Provider
	CommandProvider *command.Provider
}

func NewCommonEnvironment(ctx *pulumi.Context) (CommonEnvironment, error) {
	randomProvider, err := random.NewProvider(ctx, "random-provider", &random.ProviderArgs{})
	if err != nil {
		return CommonEnvironment{}, err
	}
	commandProvider, err := command.NewProvider(ctx, "command-provider", &command.ProviderArgs{})
	if err != nil {
		return CommonEnvironment{}, err
	}
	env := CommonEnvironment{
		Ctx:             ctx,
		InfraConfig:     sdkconfig.New(ctx, DDInfraConfigNamespace),
		AgentConfig:     sdkconfig.New(ctx, DDAgentConfigNamespace),
		CommonNamer:     namer.NewNamer(ctx, ""),
		RandomProvider:  randomProvider,
		CommandProvider: commandProvider,
	}
	ctx.Log.Debug(fmt.Sprintf("agent version: %s", env.AgentVersion()), nil)
	ctx.Log.Debug(fmt.Sprintf("deploy: %v", env.AgentDeploy()), nil)
	ctx.Log.Debug(fmt.Sprintf("full image path: %v", env.AgentFullImagePath()), nil)
	return env, nil
}

// Infra namespace
func (e *CommonEnvironment) InfraEnvironmentNames() []string {
	envsStr := e.InfraConfig.Require(DDInfraEnvironment)
	return strings.Split(envsStr, multiValueSeparator)
}

func (e *CommonEnvironment) InfraOSFamily() string {
	return e.InfraConfig.Get(DDInfraOSFamily)
}

func (e *CommonEnvironment) KubernetesVersion() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraKubernetesVersion, "1.23")
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
	return e.GetBoolWithDefault(e.AgentConfig, DDAgentDeployParamName, true)
}

func (e *CommonEnvironment) AgentVersion() string {
	return e.AgentConfig.Get(DDAgentVersionParamName)
}

func (e *CommonEnvironment) AgentFullImagePath() string {
	return e.AgentConfig.Get(DDAgentFullImagePathParamName)
}

func (e *CommonEnvironment) AgentAPIKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDAgentAPIKeyParamName)
}

func (e *CommonEnvironment) AgentAPPKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDAgentAPPKeyParamName)
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
		return strings.Split(val, multiValueSeparator)
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
