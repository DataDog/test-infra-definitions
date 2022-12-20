package ec2vm

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/agentinstall"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/common"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/vm/os"
)

type Params struct {
	name                       string
	ami                        string
	instanceType               string
	keyPair                    string
	userData                   string
	os                         os.OS
	env                        aws.Environment
	arch                       os.Architecture
	optionalAgentInstallParams *agentinstall.Params
}

func newParams(env aws.Environment, options ...func(*Params) error) (*Params, error) {
	params := &Params{
		keyPair: env.DefaultKeyPairName(),
		env:     env,
	}

	// By default use Ubuntu
	return common.ApplyOption(params, WithOS(os.UbuntuOS, os.AMD64Arch), options)
}

// WithOS sets the instance type and the AMI.
func WithOS(osType os.OSType, arch os.Architecture) func(*Params) error {
	return func(p *Params) error {
		var err error
		var os = os.GetOS(p.env, osType)

		p.instanceType = os.GetDefaultInstanceType(arch)
		p.arch = arch
		p.os = os
		p.ami, err = os.GetAMI(arch)
		if err != nil {
			return fmt.Errorf("cannot find AMI for %v (%v): %v", osType, arch, err)
		}

		return nil
	}
}

// WithAMI set the AMI. `arch` and `osType` must match the AMI requirements.
func WithAMI(ami string, arch os.Architecture, osType os.OSType) func(*Params) error {
	return func(p *Params) error {
		p.ami = ami
		p.os = os.GetOS(p.env, osType)
		p.arch = arch
		return nil
	}
}

func WithInstanceType(instanceType string) func(*Params) error {
	return func(p *Params) error {
		p.instanceType = instanceType
		return nil
	}
}

// WithUserData set the userdata for the EC2 instance. User data contains commands that are run at the startup of the instance.
func WithUserData(userData string) func(*Params) error {
	return func(p *Params) error {
		p.userData = userData
		return nil
	}
}

// WithHostAgent installs an Agent on this EC2 instance. By default use with agentinstall.WithLatest().
func WithHostAgent(apiKey string, options ...func(*agentinstall.Params) error) func(*Params) error {
	return func(p *Params) error {
		var err error
		p.optionalAgentInstallParams, err = agentinstall.NewParams(apiKey, options...)
		return err
	}
}

func WithName(name string) func(*Params) error {
	return func(p *Params) error {
		p.name = name
		return nil
	}
}
