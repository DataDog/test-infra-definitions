package ec2

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	AMD64Arch = "x86_64"
	ARM64Arch = "arm64"
)

// Latest 22.04 (jammy)
func LatestUbuntuAMI(ctx *pulumi.Context, arch string) (string, error) {
	return SearchAMI(ctx, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", arch)
}

func SearchAMI(ctx *pulumi.Context, owner, name, arch string) (string, error) {
	image, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		MostRecent: pulumi.BoolRef(true),
		Filters: []ec2.GetAmiFilter{
			{
				Name: "name",
				Values: []string{
					name,
				},
			},
			{
				Name: "virtualization-type",
				Values: []string{
					"hvm",
				},
			},
			{
				Name: "architecture",
				Values: []string{
					arch,
				},
			},
		},
		Owners: []string{
			owner,
		},
	}, nil)
	if err != nil {
		return "", err
	}

	if image == nil {
		return "", fmt.Errorf("unable to find AMI with owner: %s, name: %s, arch: %s", owner, name, arch)
	}

	return image.Id, nil
}
