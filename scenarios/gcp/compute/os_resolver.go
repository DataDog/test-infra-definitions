package compute

import (
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
)

type imageResolveFunc func(e gcp.Environment, osInfo os.Descriptor) (string, error)

var imageResolvers = map[os.Flavor]imageResolveFunc{
	os.Ubuntu: resolveUbuntuImage,
}

func resolveUbuntuImage(e gcp.Environment, osInfo os.Descriptor) (string, error) {
	if osInfo.Version == "" {
		osInfo.Version = os.UbuntuDefault.Version
	}

	switch osInfo.Version {
	case os.Ubuntu2204.Version:
		return "ubuntu-2204-jammy-v20240904", nil
	default:
		return "", nil
	}
}
