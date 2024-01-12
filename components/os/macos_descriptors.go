package os

// Implements commonly used descriptors for easier usage
var (
	MacOSDefault = MacOSSonoma
	MacOSSonoma  = NewDescriptorWithArch(MacosOS, "sonoma", ARM64Arch)
)
