package ec2

import (
	"errors"
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ec2"
)

type amiInformation struct {
	id          string
	defaultUser string
	readyFunc   command.ReadyFunc
}

type amiResolverFunc func(aws.Environment, *os.Descriptor) (string, error)

var amiResolvers = map[os.Flavor]amiResolverFunc{
	os.WindowsServer:  resolveWindowsAMI,
	os.Ubuntu:         resolveUbuntuAMI,
	os.AmazonLinux:    resolveAmazonLinuxAMI,
	os.AmazonLinuxECS: resolveAmazonLinuxECSAMI,
	os.Debian:         resolveDebianAMI,
	os.RedHat:         resolveRedHatAMI,
	os.Suse:           resolveSuseAMI,
	os.Fedora:         resolveFedoraAMI,
	os.CentOS:         resolveCentOSAMI,
	os.RockyLinux:     resolveRockyLinuxAMI,
	os.MacosOS:        resolveMacosAMI,
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

// resolveOS returns the AMI ID for the given OS.
// Note that you may get this error in some cases:
// OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe
// This means that you need to go to the AWS Marketplace and accept the terms of the AMI.
func resolveOS(e aws.Environment, vmArgs *vmArgs) (*amiInformation, error) {
	if vmArgs.ami == "" {
		var err error
		vmArgs.ami, err = amiResolvers[vmArgs.osInfo.Flavor](e, vmArgs.osInfo)
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

// Will resolve the AMI for the target OS given the platforms.json file
func resolvePinnedAMI(osInfo *os.Descriptor) (string, error) {
	var arch string
	switch osInfo.Architecture {
	case os.AMD64Arch:
		arch = "x86_64"
	case os.ARM64Arch:
		arch = "arm64"
	default:
		return "", fmt.Errorf("architecture %s is not supported for %s", osInfo.Architecture, osInfo.Flavor.String())
	}

	ami, err := aws.GetAMI(osInfo.Flavor.String(), arch, osInfo.Version)
	if err != nil {
		return "", err
	}

	return ami, nil
}

func resolveWindowsAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Architecture == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Windows")
	}
	if osInfo.Version == "" {
		osInfo.Version = os.WindowsDefault.Version
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/ami-windows-latest/Windows_Server-%s-English-Full-Base", osInfo.Version))
}

func resolveAmazonLinuxAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.AmazonLinuxDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveAmazonLinuxECSAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.AmazonLinuxECSDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveUbuntuAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.UbuntuDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveDebianAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.DebianDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveRedHatAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.RedHatDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveSuseAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.SuseDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveFedoraAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Architecture == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Fedora")
	}

	if osInfo.Version == "" {
		osInfo.Version = os.FedoraDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveCentOSAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.CentOSDefault.Version
	}

	return resolvePinnedAMI(osInfo)
}

func resolveRockyLinuxAMI(_ aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version != "" {
		return "", fmt.Errorf("cannot set version for Rocky Linux")
	}

	osInfo.Version = "default"

	return resolvePinnedAMI(osInfo)
}

func resolveMacosAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.MacOSSonoma.Version
	}

	return resolvePinnedAMI(osInfo)
}
