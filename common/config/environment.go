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
	ddInfraEnvironment       = "env"
	ddInfraKubernetesVersion = "kubernetesVersion"

	// Agent Namespace
	ddAgentDeployParamName        = "deploy"
	ddAgentVersionParamName       = "version"
	ddAgentFullImagePathParamName = "fullImagePath"
	DDAgentAPIKeyParamName        = "apiKey"
	ddAgentAPPKeyParamName        = "appKey"
)

var randomProvider *random.Provider
var commandProvider *command.Provider

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
		AgentConfig: sdkconfig.New(ctx, DDAgentConfigNamespace),
		CommonNamer: namer.NewNamer(ctx, ""),
	}
}

// Infra namespace
func (e *CommonEnvironment) InfraEnvironmentNames() []string {
	envsStr := e.InfraConfig.Require(ddInfraEnvironment)
	return strings.Split(envsStr, multiValueSeparator)
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
	return e.GetBoolWithDefault(e.AgentConfig, ddAgentDeployParamName, true)
}

func (e *CommonEnvironment) AgentVersion() string {
	return e.AgentConfig.Get(ddAgentVersionParamName)
}

func (e *CommonEnvironment) AgentFullImagePath() string {
	return e.AgentConfig.Get(ddAgentFullImagePathParamName)
}

func (e *CommonEnvironment) AgentAPIKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDAgentAPIKeyParamName)
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

func (e *CommonEnvironment) CommandProvider(namer namer.Namer, name string) (*command.Provider, error) {
	var err error
	if commandProvider != nil {
		return commandProvider, nil
	}
	commandProvider, err = command.NewProvider(e.Ctx, namer.ResourceName("provider", name), &command.ProviderArgs{})
	return commandProvider, err
}

func (e *CommonEnvironment) RandomProvider(namer namer.Namer, name string) (*random.Provider, error) {
	var err error
	if randomProvider != nil {
		return randomProvider, nil
	}
	randomProvider, err = random.NewProvider(e.Ctx, namer.ResourceName("provider", name), &random.ProviderArgs{})
	return randomProvider, err
}

type Environment interface {
	DefaultInstanceType() string
	DefaultARMInstanceType() string
	GetCommonEnvironment() *CommonEnvironment
	DefaultPrivateKeyPath() string
	DefaultPrivateKeyPassword() string
}
