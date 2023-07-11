//nolint:revive
package os

// Implements commonly used descriptors for easier usage
var (
	UbuntuDefault = Ubuntu_2204
	Ubuntu_2204   = NewDescriptor(Ubuntu, "22.04")

	DebianDefault = Debian_12
	Debian_12     = NewDescriptor(Debian, "12.0")

	AmazonLinuxDefault = AmazonLinux_2023
	AmazonLinux_2023   = NewDescriptor(AmazonLinux, "2023")
	AmazonLinux_2      = NewDescriptor(AmazonLinux, "2")

	AmazonLinuxECSDefault = AmazonLinuxECS_2023
	AmazonLinuxECS_2023   = NewDescriptor(AmazonLinuxECS, "2023")
	AmazonLinuxECS_2      = NewDescriptor(AmazonLinuxECS, "2")

	RedHatDefault = RedHat_9
	RedHat_9      = NewDescriptor(RedHat, "9.1")

	SuseDefault = Suse_15
	Suse_15     = NewDescriptor(Suse, "15-sp4")

	FedoraDefault = Fedora_37
	Fedora_37     = NewDescriptor(Fedora, "37")

	CentOSDefault = CentOS_7
	CentOS_7      = NewDescriptor(CentOS, "7")
)
