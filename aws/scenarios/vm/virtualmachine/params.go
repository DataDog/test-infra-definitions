package virtualmachine

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
)

type Architecture string
type OS int

const (
	AMD64Arch = Architecture("x86_64-2")
	ARM64Arch = Architecture("arm64")
	WindowsOS = iota
	LinuxOS   = iota
	MacOS     = iota
)

type Params struct {
	ami          string
	arch         Architecture
	instanceType string
	keyPair      string
	userData     string
	os           OS
	env          aws.Environment
}

func WithAmi(ami string, arch Architecture) func(*Params) error {
	return func(p *Params) error {
		p.ami = ami
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

func WithUserData(userData string) func(*Params) error {
	return func(p *Params) error {
		p.userData = userData
		return nil
	}
}

func WithOS(os OS, arch Architecture) func(*Params) error {
	return func(p *Params) error {
		var owner = ""
		var name = ""

		if arch == AMD64Arch {
			p.instanceType = "t3.large"
		} else {
			p.instanceType = "m6g.medium"
		}
		switch os {
		case WindowsOS:
			owner = "801119661308"
			name = "Windows_Server-2022-English-Full-Base-*"
		case LinuxOS:
			owner = "099720109477"
			name = "ubuntu/images/hvm-ssd/ubuntu-jammy-*"
		case MacOS:
			owner = "628277914472"
			name = "amzn-ec2-macos-13.*"
			if arch == AMD64Arch {
				arch = "x86_64_mac"
			} else {
				arch = "arm64_mac"
			}
		}
		p.arch = arch

		var err error
		p.ami, err = ec2.SearchAMI(p.env, owner, name, string(arch))
		if err != nil {
			return fmt.Errorf("*** cannot find AMI for %v %v *** : %v", os, arch, err)
		}

		return nil
	}
}
