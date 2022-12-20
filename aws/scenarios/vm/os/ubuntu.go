package os

import (
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
)

type ubuntu struct{ unix }

func (*ubuntu) GetSSHUser() string { return "ubuntu" }

func (u *ubuntu) GetAMI(arch Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", string(arch))
}

func (*ubuntu) GetServiceManager() *serviceManager {
	return &serviceManager{restartCmd: []string{"sudo service datadog-agent restart"}}
}

func (*ubuntu) GetOSType() OSType { return UbuntuOS }
