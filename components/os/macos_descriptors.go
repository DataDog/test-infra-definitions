package os

// Implements commonly used descriptors for easier usage
// See platforms.json for the AMIs used for each OS
var (
	MacOSDefault = MacOSSonoma
	MacOSSonoma  = NewDescriptor(MacosOS, "sonoma")
)
