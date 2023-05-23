package os

import (
	"fmt"

	commonos "github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
)

type OS interface {
	commonos.OS
	GetAMIArch(arch commonos.Architecture) string
	GetTenancy() string
}

type Type int

const (
	WindowsOS Type = iota
	UbuntuOS       = iota
	// MacosOS            = iota // Not yet supported
	AmazonLinuxOS = iota
	DebianOS      = iota
	RedHatOS      = iota
	SuseOS        = iota
	FedoraOS      = iota
)

func GetOS(env aws.Environment, osType Type) (OS, error) {
	switch osType {
	case WindowsOS:
		return newWindows(env), nil
	case UbuntuOS:
		return newUbuntu(env), nil
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
	default:
		return nil, fmt.Errorf("cannot find environment: %v", osType)
	}
}
