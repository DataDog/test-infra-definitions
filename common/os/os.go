package os

type Architecture string

const (
	AMD64Arch = Architecture("x86_64")
	ARM64Arch = Architecture("arm64")
)

type OSType int

const (
	WindowsOS OSType = iota
	UbuntuOS         = iota
	MacosOS          = iota
)

type OS interface {
	GetImage(Architecture) (string, error)
	GetDefaultInstanceType(Architecture) string
	GetServiceManager() *serviceManager
	GetConfigPath() string
	GetOSType() OSType
}
