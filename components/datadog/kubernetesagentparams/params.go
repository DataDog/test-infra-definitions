package kubernetesagentparams

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"gopkg.in/yaml.v3"

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
	HelmValues pulumi.AssetOrArchiveArray
	// PulumiDependsOn is a list of resources to depend on.
	PulumiResourceOptions []pulumi.ResourceOption
	// FakeIntake is the fake intake to use for the agent installation.
	FakeIntake *fakeintake.Fakeintake
	// DeployWindows is a flag to deploy the agent on Windows.
	DeployWindows bool
	// DisableLogsContainerCollectAll is a flag to disable collection of logs from all containers by default.
	DisableLogsContainerCollectAll bool
	// DualShipping is a flag to enable dual shipping.
	DisableDualShipping bool
	// OTelAgent is a flag to deploy the OTel agent.
	OTelAgent bool
	// OTelConfig is the OTel configuration to use for the agent installation.
	OTelConfig string
	// GKEAutopilot is a flag to deploy the agent with only GKE Autopilot compatible values.
	GKEAutopilot bool
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	version := &Params{
		Namespace: defaultAgentNamespace,
	}
	return common.ApplyOption(version, options)
}

// WithClusterName sets the name of the cluster. Should only be used if you know what you are doing. Must no be necessary in most cases.
// Mainly used to set the clusterName when the agent is installed on Kind clusters. Because the agent is not able to detect the cluster name.
// It takes a pulumi.StringOutput as input to be able to use the pulumi output of the cluster name.
func WithClusterName(clusterName pulumi.StringOutput) func(*Params) error {
	return func(p *Params) error {
		values := pulumi.Sprintf(`
datadog:
  clusterName: %s
`, clusterName)

		p.HelmValues = append(p.HelmValues, values.ApplyT(func(clusterName string) (pulumi.Asset, error) {
			return pulumi.NewStringAsset(clusterName), nil
		}).(pulumi.AssetOutput))
		return nil
	}
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
func WithHelmValues(values string) func(*Params) error {
	return func(p *Params) error {
		p.HelmValues = append(p.HelmValues, pulumi.NewStringAsset(values))
		return nil
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

// WithoutDualShipping disables dual shipping. By default the agent is configured to send data to ddev and to the fakeintake.
// With that flag data will be sent only to the fakeintake.
func WithoutDualShipping() func(*Params) error {
	return func(p *Params) error {
		p.DisableDualShipping = true
		return nil
	}
}

func WithOTelAgent() func(*Params) error {
	return func(p *Params) error {
		p.OTelAgent = true
		otelCollectorEnabledValues := `
datadog:
  otelCollector:
    enabled: true`

		p.HelmValues = append(p.HelmValues, pulumi.NewStringAsset(otelCollectorEnabledValues))
		return nil
	}
}

func WithOTelConfig(config string) func(*Params) error {
	return func(p *Params) error {
		var err error
		p.OTelConfig, err = utils.MergeYAML(p.OTelConfig, config)
		return err
	}
}

func WithGKEAutopilot() func(*Params) error {
	return func(p *Params) error {
		p.GKEAutopilot = true
		return nil
	}
}

func WithTags(tags []string) func(*Params) error {
	return func(p *Params) error {
		tagsYAML, err := yaml.Marshal(tags)
		if err != nil {
			return err
		}
		tagsHelmValues := fmt.Sprintf(`
datadog:
  tags:
%s
  dogstatsd:
    tags:
%s
`, utils.IndentMultilineString(string(tagsYAML), 4), utils.IndentMultilineString(string(tagsYAML), 6))
		return WithHelmValues(tagsHelmValues)(p)
	}
}
