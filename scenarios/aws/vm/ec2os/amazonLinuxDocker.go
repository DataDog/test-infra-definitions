package ec2os

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type amazonLinuxDocker struct {
	*amazonLinux
}

func newAmazonLinuxDocker(env aws.Environment) *amazonLinuxDocker {
	return &amazonLinuxDocker{
		amazonLinux: newAmazonLinux(env),
	}
}

func (u *amazonLinuxDocker) GetImage(arch os.Architecture) (string, error) {
	return ec2.GetLatestAMI(u.env, arch,
		"/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id",
		"/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id")
}
