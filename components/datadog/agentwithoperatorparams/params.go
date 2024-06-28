package agentwithoperatorparams

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
)

type Params struct {
	PulumiResourceOptions []pulumi.ResourceOption

	Namespace        string
	FakeIntake       *fakeintake.Fakeintake
	DDAConfig        string
	KubeletTLSVerify bool
}

type Option = func(*Params) error

func NewParams(options ...Option) (*Params, error) {
	version := &Params{
		Namespace: "datadog",
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

// WithTLSKubeletVerify toggles kubelet TLS verification.
func WithTLSKubeletVerify(verify bool) func(*Params) error {
	return func(p *Params) error {
		p.KubeletTLSVerify = verify
		return nil
	}
}

// WithPulumiResourceOptions sets the resources to depend on.
func WithPulumiResourceOptions(resources ...pulumi.ResourceOption) func(*Params) error {
	return func(p *Params) error {
		p.PulumiResourceOptions = append(p.PulumiResourceOptions, resources...)
		return nil
	}
}

// WithDDAConfig configures the DatadogAgent resource.
func WithDDAConfig(config string) func(*Params) error {
	return func(p *Params) error {
		p.DDAConfig = p.DDAConfig + config
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
