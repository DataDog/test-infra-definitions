package os

// Implements commonly used descriptors for easier usage
var (
	WindowsDefault    = WindowsServer2022
	WindowsServer2022 = NewDescriptor(WindowsServer, "2022")
	WindowsServer2019 = NewDescriptor(WindowsServer, "2019")
)
