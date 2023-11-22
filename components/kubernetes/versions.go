package kubernetes

type kindConfig struct {
	kindVersion string
	nodeImage   string
}

// Source: https://github.com/kubernetes-sigs/kind/releases
var kubeToKindVersion = map[string]kindConfig{
	"1.27": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.27.3",
	},
	"1.26": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.26.6",
	},
	"1.25": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.25.11",
	},
	"1.24": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.24.15",
	},
	"1.23": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.23.17",
	},
	"1.22": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.22.17",
	},
	"1.21": {
		kindVersion: "v0.20.0",
		nodeImage:   "kindest/node:v1.21.14",
	},
	"1.20": {
		kindVersion: "v0.17.0",
		nodeImage:   "kindest/node:v1.20.15",
	},
	"1.19": {
		kindVersion: "v0.17.0",
		nodeImage:   "kindest/node:v1.19.16",
	},
}

func kubeSupportedVersions() []string {
	var versions = make([]string, 0)

	for kubeVersion := range kubeToKindVersion {
		versions = append(versions, kubeVersion)
	}

	return versions
}
