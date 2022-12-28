package vmconfig

type Kernel struct {
	Dir string `json:"dir"`
	Tag string `json:"tag"`
}

type Image struct {
	ImageName      string `json:"image_path"`
	ImageSourceURI string `json:"image_uri"`
}

type VMSet struct {
	Name    string   `json:"name"`
	Kernels []Kernel `json:"kernels"`
	VCpu    []int    `json:"vcpu"`
	Memory  []int    `json:"memory"`
	Img     Image    `json:"image"`
}

type Config struct {
	Workdir string  `json:"workdir"`
	VMSets  []VMSet `json:"vmsets"`
	SSHKey  string  `json:"sshkey,omitempty"`
	SSHUser string  `json:"ssh_user,omitempty"`
}
