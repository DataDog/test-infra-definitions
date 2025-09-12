package ec2

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
)

type amiInformation struct {
	id          string
	defaultUser string
	readyFunc   command.ReadyFunc
}

var defaultUsers = map[os.Flavor]string{
	os.WindowsServer:  "Administrator",
	os.Ubuntu:         "ubuntu",
	os.AmazonLinux:    "ec2-user",
	os.AmazonLinuxECS: "ec2-user",
	os.Debian:         "admin",
	os.RedHat:         "ec2-user",
	os.Suse:           "ec2-user",
	os.Fedora:         "fedora",
	os.CentOS:         "centos",
	os.RockyLinux:     "cloud-user",
	os.MacosOS:        "ec2-user",
}

// Returns the default version for the given flavor
func getDefaultVersion(flavor os.Flavor) (string, error) {
	if version, ok := os.LinuxDescriptorsDefault[flavor]; ok {
		return version.Version, nil
	}
	if version, ok := os.WindowsDescriptorsDefault[flavor]; ok {
		return version.Version, nil
	}
	if version, ok := os.MacOSDescriptorsDefault[flavor]; ok {
		return version.Version, nil
	}

	return "", fmt.Errorf("no default version found for flavor %s, this flavor should be added to the default descriptors", flavor)
}

// resolveOS returns the AMI ID for the given OS.
// Note that you may get this error in some cases:
// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
// This means that you need to go to the AWS Marketplace and accept the terms of the AMI.
func resolveOS(e aws.Environment, vmArgs *vmArgs) (*amiInformation, error) {
	if vmArgs.ami == "" {
		var err error
		// If the version is not set, use the default version of this flavor
		if vmArgs.osInfo.Version == "" {
			vmArgs.osInfo.Version, err = getDefaultVersion(vmArgs.osInfo.Flavor)
			if err != nil {
				return nil, err
			}
		}
		vmArgs.ami, err = aws.GetAMI(vmArgs.osInfo)
		if err != nil {
			return nil, err
		}
	}
	fmt.Printf("Using AMI %s\n for stack %s\n", vmArgs.ami, e.Ctx().Stack())

	amiInfo := &amiInformation{
		id:          vmArgs.ami,
		defaultUser: defaultUsers[vmArgs.osInfo.Flavor],
	}

	switch vmArgs.osInfo.Family() { // nolint:exhaustive
	case os.LinuxFamily:
		if vmArgs.osInfo.Version == os.AmazonLinux2018.Version && vmArgs.osInfo.Flavor == os.AmazonLinux2018.Flavor {
			amiInfo.readyFunc = command.WaitForSuccessfulConnection
		} else {
			amiInfo.readyFunc = command.WaitForCloudInit
		}
	case os.WindowsFamily, os.MacOSFamily:
		amiInfo.readyFunc = command.WaitForSuccessfulConnection
	default:
		return nil, fmt.Errorf("unsupported OS family %v", vmArgs.osInfo.Family())
	}

	return amiInfo, nil
}
