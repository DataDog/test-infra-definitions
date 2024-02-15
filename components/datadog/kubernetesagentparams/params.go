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
//   - [WithImageTag]
//   - [WithRepository]
//   - [WithFullImagePath]
//   - [WithPulumiDependsOn]
//   - [WithDeployWindows]
//   - [WithHelmValues]
//   - [WithNamespace]
//   - [WithHostName]
//	 - [WithFakeintake]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

type Params struct {
	// FullImagePath is the full path of the docker agent image to use.
	// It has priority over ImageTag and Repository.
	FullImagePath string
	// ImageTag is the docker agent image tag to use.
	ImageTag string
	// Repository is the docker repository to use.
	Repository string
	// Namespace is the namespace to deploy the agent to.
	Namespace string
	// HelmValues is the Helm values to use for the agent installation.
	HelmValues pulumi.AssetOrArchiveArray
	// PulumiDependsOn is a list of resources to depend on.
	PulumiDependsOn []pulumi.ResourceOption
	// FakeIntake is the fake intake to use for the agent installation.
	FakeIntake *fakeintake.Fakeintake
	// DeployWindows is a flag to deploy the agent on Windows.
	DeployWindows bool
}

type Option = func(*Params) error

func NewParams(e *config.CommonEnvironment, options ...Option) (*Params, error) {
	version := &Params{
		Namespace: defaultAgentNamespace,
	}

	if e.PipelineID() != "" && e.CommitSHA() != "" {
		options = append(options, WithFullImagePath(utils.BuildDockerImagePath("669783387624.dkr.ecr.us-east-1.amazonaws.com/agent", fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))))
	}

	return common.ApplyOption(version, options)
}

// WithImageTag sets the agent image tag to use.
func WithImageTag(agentImageTag string) func(*Params) error {
	return func(p *Params) error {
		p.ImageTag = agentImageTag
		return nil
	}
}

// WithRepository sets the repository to use for the agent installation.
func WithRepository(repository string) func(*Params) error {
	return func(p *Params) error {
		p.Repository = repository
		return nil
	}
}

// WithFullImagePath sets the full path of the agent image to use.
func WithFullImagePath(fullImagePath string) func(*Params) error {
	return func(p *Params) error {
		p.FullImagePath = fullImagePath
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
		p.PulumiDependsOn = resources
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

// WithHelmValues sets the Helm values to use for the agent installation.
func WithHelmValues(values string) func(*Params) error {
	return func(p *Params) error {
		p.HelmValues = pulumi.AssetOrArchiveArray{pulumi.NewStringAsset(values)}
		return nil
	}
}

// WithFakeintake configures the Agent to use the given fake intake.
func WithFakeintake(fakeintake *fakeintake.Fakeintake) func(*Params) error {
	return func(p *Params) error {
		p.FakeIntake = fakeintake
		return nil
	}
}
