package ec2os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
)

type rockyLinux struct {
	*unix
	*os.Unix
	env aws.Environment
}

var _ OS = &rockyLinux{}

func newRockyLinux(env aws.Environment) *rockyLinux {
	return &rockyLinux{
		unix: newUnix(&env),
		env:  env,
		Unix: os.NewUnix(),
	}
}
func (*rockyLinux) GetSSHUser() string { return "cloud-user" }

func (u *rockyLinux) GetImage(arch os.Architecture) (string, error) {
	switch arch {
	case os.AMD64Arch:
		// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
		return "ami-071db23a8a6271e2c", nil
	case os.ARM64Arch:
		// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
		return "ami-0a22577ee769ab5b0", nil
	default:
		return "", fmt.Errorf("%v is not supported for Rocky Linux", arch)
	}
}

func (*rockyLinux) GetServiceManager() *os.ServiceManager {
	return os.NewSystemCtlServiceManager()
}

func (*rockyLinux) CreatePackageManager(runner *command.Runner) (command.PackageManager, error) {
	return newYumManager(runner), nil
}
