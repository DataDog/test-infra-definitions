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
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245",
	},
	"1.28": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.28.7@sha256:9bc6c451a289cf96ad0bbaf33d416901de6fd632415b076ab05f5fa7e4f65c58",
	},
	"1.27": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.27.11@sha256:681253009e68069b8e01aad36a1e0fa8cf18bb0ab3e5c4069b2e65cafdd70843",
	},
	"1.26": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.26.14@sha256:5d548739ddef37b9318c70cb977f57bf3e5015e4552be4e27e57280a8cbb8e4f",
	},
	"1.25": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.25.16@sha256:e8b50f8e06b44bb65a93678a65a26248fae585b3d3c2a669e5ca6c90c69dc519",
	},
	"1.24": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.24.17@sha256:bad10f9b98d54586cba05a7eaa1b61c6b90bfc4ee174fdc43a7b75ca75c95e51",
	},
	"1.23": {
		kindVersion:      "v0.22.0",
		nodeImageVersion: "v1.23.17@sha256:14d0a9a892b943866d7e6be119a06871291c517d279aedb816a4b4bc0ec0a5b3",
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
	versions := make([]string, 0, len(kubeToKindVersion))

	for kubeVersion := range kubeToKindVersion {
		versions = append(versions, kubeVersion)
	}

	return versions
}
