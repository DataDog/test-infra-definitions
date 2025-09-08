package aws

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Handles AMIs for all OSes

//go:embed platforms.json
var platformsJSON []byte

// map[os][arch][version] = ami (e.g. map[ubuntu][x86_64][22.04] = "ami-01234567890123456")
var platformsJSONMap map[string]map[string]map[string]string = nil

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

func GetAMI(os string, arch string, version string) (string, error) {
	platformsJSONMap, err := GetPlatformsJSONMap()
	if err != nil {
		return "", err
	}

	if _, ok := platformsJSONMap[os]; !ok {
		return "", fmt.Errorf("os '%s' not found in platforms.json", os)
	}
	if _, ok := platformsJSONMap[os][arch]; !ok {
		return "", fmt.Errorf("arch '%s' not found in platforms.json", arch)
	}
	if _, ok := platformsJSONMap[os][arch][version]; !ok {
		return "", fmt.Errorf("version '%s' not found in platforms.json", version)
	}

	return platformsJSONMap[os][arch][version], nil
}
