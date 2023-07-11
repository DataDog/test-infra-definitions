//nolint:revive
package os

// Implements commonly used descriptors for easier usage
var (
	WindowsDefault     = WindowsServer_2022
	WindowsServer_2022 = NewDescriptor(WindowsServer, "2022")
	WindowsServer_2019 = NewDescriptor(WindowsServer, "2019")
)
