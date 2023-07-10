package ec2os

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type ubuntuDocker struct {
	os.Ubuntu
	*unix
	env aws.Environment
}

func newUbuntuDocker(env aws.Environment) *ubuntuDocker {
	return &ubuntuDocker{
		Ubuntu: *os.NewUbuntu(&env),
		unix:   &unix{},
		env:    env,
	}
}
func (*ubuntuDocker) GetSSHUser() string { return "ubuntu" }

func (u *ubuntuDocker) GetImage(arch os.Architecture) (string, error) {
	switch arch {
	case os.AMD64Arch:
		return ec2.SearchAMI(u.env, "679593333241", "Docker CE and Docker Compose Image*", string(arch))
	case os.ARM64Arch:
		return "", errors.New("ARM64 is not supported for Ubuntu with Docker")
	default:
		return "", fmt.Errorf("%v is not supported for Ubuntu with Docker", arch)
	}
}
