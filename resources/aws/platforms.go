package aws

import (
	_ "embed"
	"encoding/json"
	"fmt"

	e2eos "github.com/DataDog/test-infra-definitions/components/os"
)

// Handles AMIs for all OSes

//go:embed platforms.json
var platformsJSON []byte

// map[os][arch][version] = ami (e.g. map[ubuntu][x86_64][22.04] = "ami-01234567890123456")
type platformsJSONMapType = map[string]map[string]map[string]string

var platformsJSONMap platformsJSONMapType

// Returns the parsed platforms.json map
func GetPlatformsJSONMap() (map[string]map[string]map[string]string, error) {
	if platformsJSONMap == nil {
		platformsJSONMap = make(map[string]map[string]map[string]string)
		err := json.Unmarshal(platformsJSON, &platformsJSONMap)
		if err != nil {
			return nil, err
		}
	}

	return platformsJSONMap, nil
}

func GetAMI(descriptor *e2eos.Descriptor) (string, error) {
	platformsJSONMap, err := GetPlatformsJSONMap()
	if err != nil {
		return "", err
	}

	if _, ok := platformsJSONMap[descriptor.Flavor.String()]; !ok {
		return "", fmt.Errorf("os '%s' not found in platforms.json", descriptor.Flavor.String())
	}
	if _, ok := platformsJSONMap[descriptor.Flavor.String()][string(descriptor.Architecture)]; !ok {
		return "", fmt.Errorf("arch '%s' not found in platforms.json", descriptor.Architecture)
	}
	if _, ok := platformsJSONMap[descriptor.Flavor.String()][string(descriptor.Architecture)][descriptor.Version]; !ok {
		return "", fmt.Errorf("version '%s' not found in platforms.json", descriptor.Version)
	}

	return platformsJSONMap[descriptor.Flavor.String()][string(descriptor.Architecture)][descriptor.Version], nil
}
