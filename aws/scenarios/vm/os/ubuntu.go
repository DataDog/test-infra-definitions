package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/DataDog/test-infra-definitions/common/os"
)

type ubuntu struct {
	os.Ubuntu
	*unix
	env aws.Environment
}

func newUbuntu(env aws.Environment) *ubuntu {
	return &ubuntu{
		Ubuntu: *os.NewUbuntu(&env),
		unix:   &unix{},
		env:    env,
	}
}
func (*ubuntu) GetSSHUser() string { return "ubuntu" }

func (u *ubuntu) GetImage(arch os.Architecture) (string, error) {
	return ec2.SearchAMI(u.env, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", string(arch))
}
