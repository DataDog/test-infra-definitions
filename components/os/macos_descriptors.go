//nolint:revive
package os

// Implements commonly used descriptors for easier usage
var (
	MacOSDefault = MacOS_Sonoma
	MacOS_Sonoma = NewDescriptorWithArch(MacosOS, "sonoma", ARM64Arch)
)
