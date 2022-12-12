package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
)

type ubuntu struct{ unix }

func (ubuntu) GetSSHUser() string { return "ubuntu" }

func (ubuntu) GetAMI(env aws.Environment, arch Architecture) (string, error) {
	return ec2.SearchAMI(env, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", string(arch))
}

func (ubuntu) GetServiceManager() *serviceManager {
	return &serviceManager{startCmd: "sudo service datadog-agent start"}
}

func (ubuntu) GetOSType() OSType { return UbuntuOS }
