package os

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Interfaces used by OS components
type PackageManager interface {
	Ensure(packageRef string, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

type ServiceManager interface {
	EnsureRunning(serviceName string, triggers pulumi.ArrayInput, opts ...pulumi.ResourceOption) (*remote.Command, error)
}

// FileManager needs to be added here as well instead of the command package

// OS is the high-level interface for an OS INSIDE Pulumi code
type OS interface {
	Descriptor() Descriptor

	Runner() *command.Runner
	FileManager() *command.FileManager
	PackageManager() PackageManager
	ServiceManger() ServiceManager
}

// os is a generic implementation of OS interface
type os struct {
	descriptor     Descriptor
	runner         *command.Runner
	fileManager    *command.FileManager
	packageManager PackageManager
	serviceManager ServiceManager
}

func (o os) Descriptor() Descriptor {
	return o.descriptor
}

func (o os) Runner() *command.Runner {
	return o.runner
}

func (o os) FileManager() *command.FileManager {
	return o.fileManager
}

func (o os) PackageManager() PackageManager {
	return o.packageManager
}

func (o os) ServiceManger() ServiceManager {
	return o.serviceManager
}

func NewOS(
	e config.CommonEnvironment,
	descriptor Descriptor,
	runner *command.Runner,
) OS {
	switch descriptor.Family() {
	case LinuxFamily:
		return newLinuxOS(e, descriptor, runner)
	case WindowsFamily:
		return newWindowsOS(e, descriptor, runner)
	case MacOSFamily:
		return newMacOS(e, descriptor, runner)
	default:
		panic("unsupported OS family")
	}
}
