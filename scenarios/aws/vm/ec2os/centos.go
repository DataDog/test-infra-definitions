package ec2os

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type centos struct {
	*unix
	*os.Centos
	env aws.Environment
}

var _ OS = &centos{}

func newCentos(env aws.Environment) *centos {
	return &centos{
		unix:   newUnix(&env),
		env:    env,
		Centos: os.NewCentos(),
	}
}
func (*centos) GetSSHUser() string { return "centos" }

func (u *centos) GetImage(arch os.Architecture) (string, error) {
	switch arch {
	case os.AMD64Arch:
		return ec2.SearchAMI(u.env, "679593333241", "CentOS-7-2111-*.x86_64*", string(arch))
	case os.ARM64Arch:
		// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
		return "", errors.New("ARM64 is not supported for CentOS")
	default:
		return "", fmt.Errorf("%v is not supported for CentOS", arch)
	}
}
