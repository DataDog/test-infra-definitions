package kubernetes

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
)

type kindConfig struct {
	kindVersion      string
	nodeImageVersion string
}

// Source: https://github.com/kubernetes-sigs/kind/releases
var kubeToKindVersion = map[string]kindConfig{
	"1.29": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.29.0@sha256:eaa1450915475849a73a9227b8f201df25e55e268e5d619312131292e324d570",
	},
	"1.28": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.28.0@sha256:b7a4cad12c197af3ba43202d3efe03246b3f0793f162afb40a33c923952d5b31",
	},
	"1.27": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72",
	},
	"1.26": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb",
	},
	"1.25": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8",
	},
	"1.24": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab",
	},
	"1.23": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.23.17@sha256:59c989ff8a517a93127d4a536e7014d28e235fb3529d9fba91b3951d461edfdb",
	},
	"1.22": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.22.17@sha256:f5b2e5698c6c9d6d0adc419c0deae21a425c07d81bbf3b6a6834042f25d4fba2",
	},
	"1.21": {
		kindVersion:      "v0.20.0",
		nodeImageVersion: "v1.21.14@sha256:8a4e9bb3f415d2bb81629ce33ef9c76ba514c14d707f9797a01e3216376ba093",
	},
	"1.20": {
		kindVersion:      "v0.17.0",
		nodeImageVersion: "v1.20.15@sha256:a32bf55309294120616886b5338f95dd98a2f7231519c7dedcec32ba29699394",
	},
	"1.19": {
		kindVersion:      "v0.17.0",
		nodeImageVersion: "v1.19.16@sha256:476cb3269232888437b61deca013832fee41f9f074f9bed79f57e4280f7c48b7",
	},
}

// getKindVersionConfig returns the kind version and the kind node image to use based on kubernetes version
func getKindVersionConfig(kubeVersion string) (*kindConfig, error) {
	kubeSemVer, err := semver.NewVersion(kubeVersion)
	if err != nil {
		return nil, err
	}

	kindVersionConfig, found := kubeToKindVersion[fmt.Sprintf("%d.%d", kubeSemVer.Major(), kubeSemVer.Minor())]
	if !found {
		return nil, fmt.Errorf("unsupported kubernetes version. Supported versions are %s", strings.Join(kubeSupportedVersions(), ", "))
	}

	return &kindVersionConfig, nil
}

// kubeSupportedVersions returns a comma-separated list of supported kubernetes versions
func kubeSupportedVersions() []string {
	var versions = make([]string, 0)

	for kubeVersion := range kubeToKindVersion {
		versions = append(versions, kubeVersion)
	}

	return versions
}
