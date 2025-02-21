package os

// Implements commonly used descriptors for easier usage
var (
	Ubuntu2404    = NewDescriptor(Ubuntu, "24.04")
	UbuntuDefault = Ubuntu2404
	Ubuntu2204    = NewDescriptor(Ubuntu, "22.04")

	DebianDefault = Debian12
	Debian12      = NewDescriptor(Debian, "12")

	AmazonLinuxDefault = AmazonLinux2023
	AmazonLinux2023    = NewDescriptor(AmazonLinux, "2023")
	AmazonLinux2       = NewDescriptor(AmazonLinux, "2")
	AmazonLinux2018    = NewDescriptor(AmazonLinux, "2018")

	AmazonLinuxECSDefault = AmazonLinuxECS2023
	AmazonLinuxECS2023    = NewDescriptor(AmazonLinuxECS, "2023")
	AmazonLinuxECS2       = NewDescriptor(AmazonLinuxECS, "2")

	RedHatDefault = RedHat9
	RedHat9       = NewDescriptor(RedHat, "9")

	SuseDefault = Suse15
	Suse15      = NewDescriptor(Suse, "15-sp4")

	FedoraDefault = Fedora40
	Fedora40      = NewDescriptor(Fedora, "40")

	CentOSDefault = CentOS7
	CentOS7       = NewDescriptor(CentOS, "7")
)
