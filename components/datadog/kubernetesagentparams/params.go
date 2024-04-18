package kubernetesagentparams

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	defaultAgentNamespace = "datadog"
)

// Params defines the parameters for the Kubernetes Agent installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithAgentFullImagePath]
//   - [WithClusterAgentFullImagePath]
//   - [WithPulumiResourceOptions]
//   - [WithDeployWindows]
//   - [WithHelmValues]
//   - [WithNamespace]
//   - [WithDeployWindows]
//   - [WithFakeintake]
//   - [WithoutLogsContainerCollectAll]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type Params struct {
	// AgentFullImagePath is the full path of the docker agent image to use.
	AgentFullImagePath string
	// ClusterAgentFullImagePath is the full path of the docker cluster agent image to use.
	ClusterAgentFullImagePath string
	// Namespace is the namespace to deploy the agent to.
	Namespace string
	// HelmValues is the Helm values to use for the agent installation.
	HelmValues string
	// PulumiDependsOn is a list of resources to depend on.
	PulumiResourceOptions []pulumi.ResourceOption
	// FakeIntake is the fake intake to use for the agent installation.
	FakeIntake *fakeintake.Fakeintake
	// DeployWindows is a flag to deploy the agent on Windows.
	DeployWindows bool
	// DisableLogsContainerCollectAll is a flag to disable collection of logs from all containers by default.
	DisableLogsContainerCollectAll bool
}

type Option = func(*Params) error

func NewParams(e config.CommonEnvironment, options ...Option) (*Params, error) {
	version := &Params{
		Namespace: defaultAgentNamespace,
	}

	if e.PipelineID() != "" && e.CommitSHA() != "" {
		options = append(options, WithAgentFullImagePath(utils.BuildDockerImagePath(fmt.Sprintf("%s/agent", e.CloudProviderEnvironment.InternalRegistry()), fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))))
		options = append(options, WithClusterAgentFullImagePath(utils.BuildDockerImagePath(fmt.Sprintf("%s/cluster-agent", e.CloudProviderEnvironment.InternalRegistry()), fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))))
	}

	return common.ApplyOption(version, options)
}

// WithAgentFullImagePath sets the full path of the agent image to use.
func WithAgentFullImagePath(fullImagePath string) func(*Params) error {
	return func(p *Params) error {
		p.AgentFullImagePath = fullImagePath
		return nil
	}
}

// WithClusterAgentFullImagePath sets the full path of the agent image to use.
func WithClusterAgentFullImagePath(fullImagePath string) func(*Params) error {
	return func(p *Params) error {
		p.ClusterAgentFullImagePath = fullImagePath
		return nil
	}
}

// WithNamespace sets the namespace to deploy the agent to.
func WithNamespace(namespace string) func(*Params) error {
	return func(p *Params) error {
		p.Namespace = namespace
		return nil
	}
}

// WithPulumiDependsOn sets the resources to depend on.
func WithPulumiResourceOptions(resources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.PulumiResourceOptions = append(p.PulumiResourceOptions, resources...)
		return nil
	}
}

// WithDeployWindows sets the flag to deploy the agent on Windows.
func WithDeployWindows() func(*Params) error {
	return func(p *Params) error {
		p.DeployWindows = true
		return nil
	}
}

// WithHelmValues adds helm values to the agent installation. If used several times, the helm values are merged together
// If the same values is defined several times the latter call will override the previous one.
// TODO: If https://github.com/pulumi/pulumi-kubernetes/pull/2963 is merged we can revert https://github.com/DataDog/test-infra-definitions/pull/779
func WithHelmValues(values string) func(*Params) error {
	return func(p *Params) error {
		var err error
		p.HelmValues, err = utils.MergeYAMLString(p.HelmValues, values)
		return err
	}
}

// WithFakeintake configures the Agent to use the given fake intake.
func WithFakeintake(fakeintake *fakeintake.Fakeintake) func(*Params) error {
	return func(p *Params) error {
		p.PulumiResourceOptions = append(p.PulumiResourceOptions, utils.PulumiDependsOn(fakeintake))
		p.FakeIntake = fakeintake
		return nil
	}
}

// WithoutLogsContainerCollectAll disables collection of logs from all containers by default.
func WithoutLogsContainerCollectAll() func(*Params) error {
	return func(p *Params) error {
		p.DisableLogsContainerCollectAll = true
		return nil
	}
}
