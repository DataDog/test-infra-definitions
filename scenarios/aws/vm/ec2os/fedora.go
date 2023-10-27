package ec2os

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type fedora struct {
	*unix
	*os.Unix
	env aws.Environment
}

func newFedora(env aws.Environment) *fedora {
	return &fedora{
		unix: newUnix(&env),
		env:  env,
		Unix: os.NewUnix(),
	}
}
func (*fedora) GetSSHUser() string { return "fedora" }

func (u *fedora) GetImage(arch os.Architecture) (string, error) {
	switch arch {
	case os.AMD64Arch:
		return ec2.SearchAMI(u.env, "125523088429", "Fedora-Cloud-Base-37-*", string(arch))
	case os.ARM64Arch:
		// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
		return "", errors.New("ARM64 is not supported for Fedora")
	default:
		return "", fmt.Errorf("%v is not supported for Fedora", arch)
	}
}

func (*fedora) GetServiceManager() *os.ServiceManager {
	return os.NewSystemCtlServiceManager()
}

func (*fedora) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newDnfManager(runner), nil
}
