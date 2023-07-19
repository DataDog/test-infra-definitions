package utils

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/os"
)

func GetArchitecture(commonEnv *config.CommonEnvironment) (os.Architecture, error) {
	var arch os.Architecture
	archStr := strings.ToLower(commonEnv.InfraOSArchitecture())
	switch archStr {
	case "x86_64":
		arch = os.AMD64Arch
	case "arm64":
		arch = os.ARM64Arch
	case "":
		arch = os.AMD64Arch // Default
	default:
		return arch, fmt.Errorf("the architecture type '%v' is not valid", archStr)
	}
	return arch, nil
}
