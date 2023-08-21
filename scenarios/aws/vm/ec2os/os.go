package ec2os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
)

type OS interface {
	os.OS
	GetAMIArch(arch os.Architecture) string
	GetTenancy() string
}

type Type int

const (
	WindowsOS           Type = iota
	UbuntuOS            Type = iota
	AmazonLinuxDockerOS Type = iota
	// MacosOS         Type   = iota // Not yet supported
	AmazonLinuxOS Type = iota
	DebianOS      Type = iota
	RedHatOS      Type = iota
	SuseOS        Type = iota
	FedoraOS      Type = iota
	CentOS        Type = iota
	RockyLinux    Type = iota
)

func GetOS(env aws.Environment, osType Type) (OS, error) {
	switch osType {
	case WindowsOS:
		return newWindows(env), nil
	case UbuntuOS:
		return newUbuntu(env), nil
	case AmazonLinuxDockerOS:
		return newAmazonLinuxDocker(env), nil
	//case MacosOS:
	//return newMacOS(env), nil // Not yet supported
	case AmazonLinuxOS:
		return newAmazonLinux(env), nil
	case DebianOS:
		return newDebian(env), nil
	case RedHatOS:
		return newRedHat(env), nil
	case SuseOS:
		return newSuse(env), nil
	case FedoraOS:
		return newFedora(env), nil
	case CentOS:
		return newCentos(env), nil
	case RockyLinux:
		return newRockyLinux(env), nil
	default:
		return nil, fmt.Errorf("cannot find environment: %v", osType)
	}
}
