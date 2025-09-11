package aws

import (
	_ "embed"
	"fmt"

	e2eos "github.com/DataDog/test-infra-definitions/components/os"
)

// Handles AMIs for all OSes

// map[os][arch][version] = ami (e.g. map[ubuntu][x86_64][22.04] = "ami-01234567890123456")
type PlatformsAMIsType = map[string]string
type PlatformsArchsType = map[string]PlatformsAMIsType
type PlatformsType = map[string]PlatformsArchsType

// All the OS descriptors / AMIs correspondance
var platforms = PlatformsType{
	"debian": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"9":  "ami-0182559468c1975fe",
			"10": "ami-0ea3c9b1a2bcdabfc",
			"11": "ami-03039e227b7b461b5",
			"12": "ami-0eef9d92ec044bc94",
		},
		"arm64": PlatformsAMIsType{
			"10": "ami-0ce37c7cbd1cfc3e8",
			"11": "ami-0c3f5b0b87f042da8",
			"12": "ami-0505441d7e1514742",
		},
	},
	"ubuntu": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"14.04": "ami-013d633d3b6cdb22c",
			"16.04": "ami-089043ae24872fe78",
			"18.04": "ami-015715c1584e7243c",
			"20.04": "ami-0ffb8e1df897204c4",
			"22.04": "ami-02e99c6973f5a184a",
			"23.04": "ami-09c5d86a379ab69a5",
			"23.10": "ami-0949b45ef274e55a1",
			"24.04": "ami-0cef91cd3a5f44601",
		},
		"arm64": PlatformsAMIsType{
			"18.04":   "ami-002c0df0f55a14f82",
			"20.04":   "ami-0ccf50ab09c6df2d4",
			"20.04.2": "ami-023f1e40b096c3ebc",
			"21.04":   "ami-044f0ceee8e885e87",
			"22.04":   "ami-0f1d4680f8b775120",
			"23.04":   "ami-05fab5da2d7fe0b0b",
			"23.10":   "ami-0dea732dd5f1da0a8",
			"24.04":   "ami-026fccd88446aa0bf",
		},
	},
	"amazon-linux": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"2-4-14":    "ami-038b3df3312ddf25d",
			"2-5-10":    "ami-06a0cd9728546d178",
			"2022-5-15": "ami-0f0f00c2d082c52ae",
			"2023":      "ami-00ca32bbc84273381",
			"2018":      "ami-07541a4f680f1ba8e",
			"2":         "ami-0023921b4fcd5382b",
		},
		"arm64": PlatformsAMIsType{
			"2-4-14":    "ami-090230ed0c6b13c74",
			"2-5-10":    "ami-09e51988f56677f44",
			"2022-5-15": "ami-0acc51c3357f26240",
			"2023":      "ami-0aa7db6294d00216f",
			"2":         "ami-00aae26e31bb072a2",
		},
	},
	"amazon-linux-ecs": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"2023": "ami-0986768eff12aa2b9",
			"2":    "ami-0293ff221e87260aa",
		},
		"arm64": PlatformsAMIsType{
			"2023": "ami-032a0cd402947954b",
			"2":    "ami-07af7b838076acdcc",
		},
	},
	"redhat": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"9":       "ami-01a52a1073599b7c8",
			"86":      "ami-031eff1ae75bb87e4",
			"86-fips": "ami-0d0fb96b595c56e03",
		},
		"arm64": PlatformsAMIsType{
			"9":  " ami-089b86d2f4d27cd98",
			"86": "ami-0238411fb452f8275",
		},
	},
	"suse": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"12":     "ami-0b0597153739840c4",
			"15-sp4": "ami-067dfda331f8296b0",
		},
		"arm64": PlatformsAMIsType{
			"15-sp4": "ami-0d446ba26bbe19573",
		},
	},
	"fedora": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"40": "ami-004f552bba0e5f64f",
		},
		"arm64": PlatformsAMIsType{
			"42": "ami-0184eee8cd4a6080b",
		},
	},
	"centos": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"610": "ami-0506f01ccb6dddeda",
			"79":  "ami-036de472bb001ae9c",
		},
		"arm64": PlatformsAMIsType{
			"79": "ami-0cb7a00afccf30559",
		},
	},
	"rocky-linux": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"92": "ami-08f362c39d03a4eb5",
		},
	},
	"windows-server": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"2025": "ami-0efee5160a1079475",
			"2022": "ami-028dc1123403bd543",
			"2019": "ami-043cf96255cd85b98",
			"2016": "ami-0fe657c1315199148",
		},
	},
	"macos": PlatformsArchsType{
		"arm64": PlatformsAMIsType{
			"sonoma": "ami-0c582a76ed8159789",
		},
	},
}

func GetAMI(descriptor *e2eos.Descriptor) (string, error) {
	if _, ok := platforms[descriptor.Flavor.String()]; !ok {
		return "", fmt.Errorf("os '%s' not found in platforms map", descriptor.Flavor.String())
	}
	if _, ok := platforms[descriptor.Flavor.String()][string(descriptor.Architecture)]; !ok {
		return "", fmt.Errorf("arch '%s' not found in platforms map", descriptor.Architecture)
	}
	if _, ok := platforms[descriptor.Flavor.String()][string(descriptor.Architecture)][descriptor.Version]; !ok {
		return "", fmt.Errorf("version '%s' not found in platforms map", descriptor.Version)
	}

	return platforms[descriptor.Flavor.String()][string(descriptor.Architecture)][descriptor.Version], nil
}
