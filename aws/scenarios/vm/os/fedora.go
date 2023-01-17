package os

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type fedora struct {
	*unix
	*commonos.Unix
	env aws.Environment
}

func newFedora(env aws.Environment) *fedora {
	return &fedora{
		unix: &unix{},
		env:  env,
		Unix: commonos.NewUnix(&env),
	}
}
func (*fedora) GetSSHUser() string { return "fedora" }

func (u *fedora) GetImage(arch commonos.Architecture) (string, error) {
	switch arch {
	case commonos.AMD64Arch:
		return ec2.SearchAMI(u.env, "679593333241", "Fedora-Cloud-Base-34-1.2*-standard-*", string(arch))
	case commonos.ARM64Arch:
		// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
		return "", errors.New("ARM64 is not supported for Fedora")
	default:
		return "", fmt.Errorf("%v is not supported for Fedora", arch)
	}
}

func (*fedora) GetServiceManager() *commonos.ServiceManager {
	return commonos.NewSystemCtlServiceManager()
}
