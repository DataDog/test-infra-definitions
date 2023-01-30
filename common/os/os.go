package os

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

type Type int

const (
	WindowsOS Type = iota
	UbuntuOS       = iota
	MacosOS        = iota
)

type OS interface {
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetServiceManager() *ServiceManager
	GetAgentConfigPath() string
	GetSSHUser() string
	GetOSType() Type
}
