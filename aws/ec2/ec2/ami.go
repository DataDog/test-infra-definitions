package ec2

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	AMD64Arch = "x86_64"
	ARM64Arch = "arm64"
)

var errAmiRootDeviceNotFound = errors.New("error Could not find root device of ami")

// Latest 22.04 (jammy)
func LatestUbuntuAMI(e aws.Environment, arch string) (string, error) {
	img, err := SearchAMI(e, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", arch)
	if err != nil {
		return "", err
	}
	return img.Id, nil
}

func LatestUbuntuAMIRootDevice(e aws.Environment, arch string) (ec2.GetAmiBlockDeviceMapping, error) {
	img, err := SearchAMI(e, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", arch)
	if err != nil {
		return ec2.GetAmiBlockDeviceMapping{}, err
	}

	for _, blockDevice := range img.BlockDeviceMappings {
		if blockDevice.DeviceName == img.RootDeviceName {
			return blockDevice, nil
		}
	}

	return ec2.GetAmiBlockDeviceMapping{}, errAmiRootDeviceNotFound
}

//func LatestUbuntuAMIRootDeviceStorageSize(e aws.Environment, arch string) (int, error) {
//	img, err := searchami(e, "099720109477", "ubuntu/images/hvm-ssd/ubuntu-jammy-*", arch)
//	if err != nil {
//		return 0, err
//	}
//
//	for blockDevice := range img.BlockDeviceMappings {
//		if strings.HasPrefix(blockDevice.VirtualName, "ephemeral") {
//			continue
//		}
//
//		if volumeSize, ok := blockDevice.Ebs["volume_size"]; ok {
//			return volumeSize, nil
//		}
//	}
//
//	return 0, errAmiRootDeviceStorageSizeNotFound
//}

func SearchAMI(e aws.Environment, owner, name, arch string) (*ec2.LookupAmiResult, error) {
	image, err := ec2.LookupAmi(e.Ctx, &ec2.LookupAmiArgs{
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
	}, pulumi.Provider(e.Provider))
	if err != nil {
		return nil, err
	}

	if image == nil {
		return nil, fmt.Errorf("unable to find AMI with owner: %s, name: %s, arch: %s", owner, name, arch)
	}

	return image, nil
}
