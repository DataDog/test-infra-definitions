package ec2os

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"golang.org/x/exp/maps"
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
	factories := getOSFactories()
	factory, ok := factories[osType]
	if !ok {
		return nil, fmt.Errorf("OS type %v is not supported", osType)
	}
	return factory(env), nil
}

func GetOSTypes() []Type {
	return maps.Keys(getOSFactories())
}

type osFactory func(aws.Environment) OS

func getOSFactories() map[Type]osFactory {
	factories := make(map[Type]osFactory)
	factories[WindowsOS] = toOSFactory(newWindows)
	factories[UbuntuOS] = toOSFactory(newUbuntu)
	factories[AmazonLinuxDockerOS] = toOSFactory(newAmazonLinuxDocker)
	factories[AmazonLinuxOS] = toOSFactory(newAmazonLinux)
	factories[DebianOS] = toOSFactory(newDebian)
	factories[RedHatOS] = toOSFactory(newRedHat)
	factories[SuseOS] = toOSFactory(newSuse)
	factories[FedoraOS] = toOSFactory(newFedora)
	factories[CentOS] = toOSFactory(newCentos)
	factories[RockyLinux] = toOSFactory(newRockyLinux)
	return factories
}

func toOSFactory[T OS](fct func(aws.Environment) T) osFactory {
	return func(env aws.Environment) OS { return fct(env) }
}
