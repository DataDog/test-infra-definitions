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
	multiValueSeparator = ","

	namerNamespace             = "common"
	DDInfraConfigNamespace     = "ddinfra"
	DDAgentConfigNamespace     = "ddagent"
	DDTestingWorkloadNamespace = "ddtestworkload"
	DDDogstatsdNamespace       = "dddogstatsd"
	DDUpdaterConfigNamespace   = "ddupdater"

	// Infra namespace
	DDInfraEnvironment                      = "env"
	DDInfraKubernetesVersion                = "kubernetesVersion"
	DDInfraOSDescriptor                     = "osDescriptor" // osDescriptor is expected in the format: <osFamily>:<osVersion>:<osArch>, see components/os/descriptor.go
	DDInfraOSImageID                        = "osImageID"
	DDInfraDeployFakeintakeWithLoadBalancer = "deployFakeintakeWithLoadBalancer"
	DDInfraExtraResourcesTags               = "extraResourcesTags"

	// Agent Namespace
	DDAgentDeployParamName               = "deploy"
	DDAgentVersionParamName              = "version"
	DDAgentPipelineID                    = "pipeline_id"
	DDAgentFullImagePathParamName        = "fullImagePath"
	DDClusterAgentVersionParamName       = "clusterAgentVersion"
	DDClusterAgentFullImagePathParamName = "clusterAgentFullImagePath"
	DDImagePullRegistryParamName         = "imagePullRegistry"
	DDImagePullUsernameParamName         = "imagePullUsername"
	DDImagePullPasswordParamName         = "imagePullPassword"
	DDAgentAPIKeyParamName               = "apiKey"
	DDAgentAPPKeyParamName               = "appKey"
	DDAgentFakeintake                    = "fakeintake"

	// Updater Namespace
	DDUpdaterParamName = "deploy"

	// Testing workload namerNamespace
	DDTestingWorkloadDeployParamName = "deploy"

	// Dogstatsd namespace
	DDDogstatsdDeployParamName        = "deploy"
	DDDogstatsdFullImagePathParamName = "fullImagePath"
)

type CommonEnvironment struct {
	providerRegistry

	Ctx         *pulumi.Context
	CommonNamer namer.Namer

	InfraConfig           *sdkconfig.Config
	AgentConfig           *sdkconfig.Config
	TestingWorkloadConfig *sdkconfig.Config
	DogstatsdConfig       *sdkconfig.Config
	UpdaterConfig         *sdkconfig.Config

	username string
}

func NewCommonEnvironment(ctx *pulumi.Context) (CommonEnvironment, error) {
	env := CommonEnvironment{
		Ctx:                   ctx,
		InfraConfig:           sdkconfig.New(ctx, DDInfraConfigNamespace),
		AgentConfig:           sdkconfig.New(ctx, DDAgentConfigNamespace),
		TestingWorkloadConfig: sdkconfig.New(ctx, DDTestingWorkloadNamespace),
		DogstatsdConfig:       sdkconfig.New(ctx, DDDogstatsdNamespace),
		UpdaterConfig:         sdkconfig.New(ctx, DDUpdaterConfigNamespace),
		CommonNamer:           namer.NewNamer(ctx, ""),
		providerRegistry:      newProviderRegistry(ctx),
	}
	// store username
	user, err := user.Current()
	if err != nil {
		return env, err
	}
	env.username = strings.ReplaceAll(user.Username, "\\", "/")

	ctx.Log.Debug(fmt.Sprintf("user name: %s", env.username), nil)
	ctx.Log.Debug(fmt.Sprintf("resource tags: %v", env.DefaultResourceTags()), nil)
	ctx.Log.Debug(fmt.Sprintf("agent version: %s", env.AgentVersion()), nil)
	ctx.Log.Debug(fmt.Sprintf("pipeline id: %s", env.PipelineID()), nil)
	ctx.Log.Debug(fmt.Sprintf("deploy: %v", env.AgentDeploy()), nil)
	ctx.Log.Debug(fmt.Sprintf("full image path: %v", env.AgentFullImagePath()), nil)
	return env, nil
}

// Infra namespace

func (e *CommonEnvironment) InfraShouldDeployFakeintakeWithLB() bool {
	return e.GetBoolWithDefault(e.InfraConfig, DDInfraDeployFakeintakeWithLoadBalancer, true)
}

func (e *CommonEnvironment) InfraEnvironmentNames() []string {
	envsStr := e.InfraConfig.Require(DDInfraEnvironment)
	return strings.Split(envsStr, multiValueSeparator)
}

func (e *CommonEnvironment) InfraOSDescriptor() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraOSDescriptor, "")
}

func (e *CommonEnvironment) InfraOSImageID() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraOSImageID, "")
}

func (e *CommonEnvironment) KubernetesVersion() string {
	return e.GetStringWithDefault(e.InfraConfig, DDInfraKubernetesVersion, "1.26")
}

func (e *CommonEnvironment) DefaultResourceTags() map[string]string {
	return map[string]string{"managed-by": "pulumi", "username": e.username}
}

func (e *CommonEnvironment) ExtraResourcesTags() map[string]string {
	tags, err := tagListToKeyValueMap(e.GetStringListWithDefault(e.InfraConfig, DDInfraExtraResourcesTags, []string{}))
	if err != nil {
		e.Ctx.Log.Error(fmt.Sprintf("error in extra resources tags : %v", err), nil)
	}
	return tags
}

func EnvVariableResourceTags() map[string]string {
	tags := map[string]string{}
	lookupVars := []string{"TEAM", "PIPELINE_ID", "CI_PIPELINE_ID"}
	for _, varName := range lookupVars {
		if val := os.Getenv(varName); val != "" {
			tags[varName] = val
		}
	}
	return tags
}

func (e *CommonEnvironment) ResourcesTags() pulumi.StringMap {
	tags := pulumi.StringMap{}

	// default tags
	extendTagsMap(tags, e.DefaultResourceTags())
	// extended resource tags
	extendTagsMap(tags, e.ExtraResourcesTags())
	// env variable tags
	extendTagsMap(tags, EnvVariableResourceTags())

	return tags
}

// Agent Namespace
func (e *CommonEnvironment) AgentDeploy() bool {
	return e.GetBoolWithDefault(e.AgentConfig, DDAgentDeployParamName, true)
}

func (e *CommonEnvironment) AgentVersion() string {
	return e.AgentConfig.Get(DDAgentVersionParamName)
}

func (e *CommonEnvironment) PipelineID() string {
	return e.AgentConfig.Get(DDAgentPipelineID)
}

func (e *CommonEnvironment) ClusterAgentVersion() string {
	return e.AgentConfig.Get(DDClusterAgentVersionParamName)
}

func (e *CommonEnvironment) AgentFullImagePath() string {
	return e.AgentConfig.Get(DDAgentFullImagePathParamName)
}

func (e *CommonEnvironment) ClusterAgentFullImagePath() string {
	return e.AgentConfig.Get(DDClusterAgentFullImagePathParamName)
}

func (e *CommonEnvironment) ImagePullRegistry() string {
	return e.AgentConfig.Get(DDImagePullRegistryParamName)
}

func (e *CommonEnvironment) ImagePullUsername() string {
	return e.AgentConfig.Require(DDImagePullUsernameParamName)
}

func (e *CommonEnvironment) ImagePullPassword() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDImagePullPasswordParamName)
}

func (e *CommonEnvironment) AgentAPIKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDAgentAPIKeyParamName)
}

func (e *CommonEnvironment) AgentAPPKey() pulumi.StringOutput {
	return e.AgentConfig.RequireSecret(DDAgentAPPKeyParamName)
}

func (e *CommonEnvironment) AgentUseFakeintake() bool {
	return e.GetBoolWithDefault(e.AgentConfig, DDAgentFakeintake, true)
}

// Testing workload namespace
func (e *CommonEnvironment) TestingWorkloadDeploy() bool {
	return e.GetBoolWithDefault(e.TestingWorkloadConfig, DDTestingWorkloadDeployParamName, true)
}

// Dogstatsd namespace
func (e *CommonEnvironment) DogstatsdDeploy() bool {
	return e.GetBoolWithDefault(e.DogstatsdConfig, DDDogstatsdDeployParamName, true)
}

func (e *CommonEnvironment) DogstatsdFullImagePath() string {
	return e.GetStringWithDefault(e.DogstatsdConfig, DDDogstatsdFullImagePathParamName, "gcr.io/datadoghq/dogstatsd")
}

// Updater namespace
func (e *CommonEnvironment) UpdaterDeploy() bool {
	return e.GetBoolWithDefault(e.UpdaterConfig, DDUpdaterParamName, false)
}

// Generic methods
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
