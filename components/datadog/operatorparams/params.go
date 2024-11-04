package operatorparams

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
)

type Params struct {
	// OperatorFullImagePath is the full path of the docker agent image to use.
	OperatorFullImagePath string
	// Namespace is the namespace to deploy the agent to.
	Namespace string
	// HelmValues is the Helm values to use for the agent installation.
	HelmValues pulumi.AssetOrArchiveArray
	// PulumiResourceOptions is a list of resources to depend on.
	PulumiResourceOptions []pulumi.ResourceOption
}

type Option = func(*Params) error

func NewParams(e config.Env, options ...Option) (*Params, error) {
	version := &Params{
		Namespace:             "datadog",
		OperatorFullImagePath: "gcr.io/datadoghq/operator:latest",
	}

	if e.PipelineID() != "" && e.CommitSHA() != "" {
		options = append(options, WithOperatorFullImagePath(utils.BuildDockerImagePath(fmt.Sprintf("%s/operator", e.InternalRegistry()), fmt.Sprintf("%s-%s", e.PipelineID(), e.CommitSHA()))))
	}

	return common.ApplyOption(version, options)
}

// WithNamespace sets the namespace to deploy the agent to.
func WithNamespace(namespace string) func(*Params) error {
	return func(p *Params) error {
		p.Namespace = namespace
		return nil
	}
}

// WithOperatorFullImagePath sets the namespace to deploy the agent to.
func WithOperatorFullImagePath(path string) func(*Params) error {
	return func(p *Params) error {
		p.OperatorFullImagePath = path
		return nil
	}
}

// WithHelmValues adds helm values to the agent installation. If used several times, the helm values are merged together
// If the same values is defined several times the latter call will override the previous one.
func WithHelmValues(values string) func(*Params) error {
	return func(p *Params) error {
		p.HelmValues = append(p.HelmValues, pulumi.NewStringAsset(values))
		return nil
	}
}
