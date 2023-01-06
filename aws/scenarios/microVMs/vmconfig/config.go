package vmconfig

type Kernel struct {
	Dir         string `json:"dir"`
	Tag         string `json:"tag"`
	ImageSource string `json:"image_source,omitempty"`
}

type Image struct {
	ImageName      string `json:"image_path,omitempty"`
	ImageSourceURI string `json:"image_uri,omitempty"`
}

type VMSet struct {
	Name    string   `json:"name"`
	Recipe  string   `json:"recipe"`
	Kernels []Kernel `json:"kernels"`
	VCpu    []int    `json:"vcpu"`
	Memory  []int    `json:"memory"`
	Img     Image    `json:"image"`
	Arch    string
}

type Config struct {
	Workdir string  `json:"workdir"`
	VMSets  []VMSet `json:"vmsets"`
	SSHKey  string  `json:"sshkey,omitempty"`
	SSHUser string  `json:"ssh_user,omitempty"`
}
