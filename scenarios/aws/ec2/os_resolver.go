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
	var paramName string
	switch osInfo.Version {
	case "", os.AmazonLinuxECS2.Version:
		paramName = fmt.Sprintf("amzn2-ami-hvm-%s-gp2", osInfo.Architecture)
	case os.AmazonLinuxECS2023.Version:
		paramName = fmt.Sprintf("al2023-ami-kernel-default-%s", osInfo.Architecture)
	case os.AmazonLinux2018.Version:
		if osInfo.Architecture != os.AMD64Arch {
			return "", fmt.Errorf("arch %s is not supported for Amazon Linux 2018", osInfo.Architecture)
		}
		return ec2.SearchAMI(e, "669783387624", "amzn-ami-2018.03.*-amazon-ecs-optimized", string(osInfo.Architecture))
	default:
		return "", fmt.Errorf("unsupported Amazon Linux version %s", osInfo.Version)
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/ami-amazon-linux-latest/%s", paramName))
}

func resolveAmazonLinuxECSAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	var paramName string
	switch osInfo.Version {
	case "", os.AmazonLinuxECSDefault.Version:
		paramName = "amazon-linux-2"
	case os.AmazonLinuxECS2023.Version:
		paramName = "amazon-linux-2023"
	default:
		return "", fmt.Errorf("unsupported Amazon Linux ECS version %s", osInfo.Version)
	}

	if osInfo.Architecture == os.ARM64Arch {
		paramName += "/arm64"
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/ecs/optimized-ami/%s/recommended/image_id", paramName))
}

func resolveUbuntuAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.UbuntuDefault.Version
	}

	paramArch := osInfo.Architecture
	if paramArch == os.AMD64Arch {
		// Override required as the architecture is x86_64 but the SSM parameter is amd64
		paramArch = "amd64"
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/canonical/ubuntu/server/%s/stable/current/%s/hvm/ebs-gp2/ami-id", osInfo.Version, paramArch))
}

func resolveDebianAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.DebianDefault.Version
	}

	paramArch := osInfo.Architecture
	if paramArch == os.AMD64Arch {
		// Override required as the architecture is x86_64 but the SSM parameter is amd64
		paramArch = "amd64"
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/debian/release/%s/latest/%s", osInfo.Version, paramArch))
}

func resolveRedHatAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.RedHatDefault.Version
	}

	return ec2.SearchAMI(e, "309956199498", fmt.Sprintf("RHEL-%s*_HVM-*-2-Hourly2-GP2", osInfo.Version), string(osInfo.Architecture))
}

func resolveSuseAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.SuseDefault.Version
	}

	return ec2.GetAMIFromSSM(e, fmt.Sprintf("/aws/service/suse/sles/%s/%s/latest", osInfo.Version, osInfo.Architecture))
}

func resolveFedoraAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Architecture == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for Fedora")
	}

	if osInfo.Version == "" {
		osInfo.Version = os.FedoraDefault.Version
	}

	return ec2.SearchAMI(e, "125523088429", fmt.Sprintf("Fedora-Cloud-Base-%s-*", osInfo.Version), string(osInfo.Architecture))
}

func resolveCentOSAMI(e aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Architecture == os.ARM64Arch {
		return "", errors.New("ARM64 is not supported for CentOS")
	}

	if osInfo.Version == "" {
		osInfo.Version = os.CentOSDefault.Version
	}

	if osInfo.Version == "7" {
		return "ami-036de472bb001ae9c", nil
	}

	return ec2.SearchAMI(e, "679593333241", fmt.Sprintf("CentOS-%s-*-*.x86_64*", osInfo.Version), string(osInfo.Architecture))
}

func resolveRockyLinuxAMI(_ aws.Environment, osInfo *os.Descriptor) (string, error) {
	if osInfo.Version != "" {
		return "", fmt.Errorf("cannot set version for Rocky Linux")
	}

	var amiID string
	switch osInfo.Architecture {
	case os.AMD64Arch:
		amiID = "ami-071db23a8a6271e2c"
	case os.ARM64Arch:
		amiID = "ami-0a22577ee769ab5b0"
	default:
		return "", fmt.Errorf("architecture %s is not supported for Rocky Linux", osInfo.Architecture)
	}

	return amiID, nil
}
