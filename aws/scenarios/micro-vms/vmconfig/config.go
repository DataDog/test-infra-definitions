package vmconfig

type Kernel struct {
	Dir string `json:"dir"`
	Tag string `json:"tag"`
}

type VMSet struct {
	Name    string   `json:"name"`
	Kernels []Kernel `json:"kernels"`
	VCpu    []int    `json:"vcpu"`
	Memory  []int    `json:"memory"`
	Image   string   `json:"image"`
}

type Config struct {
	Workdir string `json:"workdir"`
	// Directory with kernel object files (e.g. `vmlinux` for linux)
	// (used for report symbolization, coverage reports and in tree modules finding, optional).
	VMSets []VMSet `json:"vmsets"`
	// Location of the disk image file.
	// Location (on the host machine) of a root SSH identity to use for communicating with
	// the virtual machine (may be empty for some VM types).
	SSHKey string `json:"sshkey,omitempty"`
	// SSH user ("root" by default).
	SSHUser string `json:"ssh_user,omitempty"`
}
