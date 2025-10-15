package aws

import (
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
	"amazon-linux": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"2-4-14":    "ami-038b3df3312ddf25d",
			"2-5-10":    "ami-06a0cd9728546d178",
			"2022-5-15": "ami-0f0f00c2d082c52ae",
			"2023":      "ami-0f4a4fa5d1c6e0704",
			"2018":      "ami-07541a4f680f1ba8e",
			"2":         "ami-0023921b4fcd5382b",
		},
		"arm64": PlatformsAMIsType{
			"2-4-14":    "ami-090230ed0c6b13c74",
			"2-5-10":    "ami-09e51988f56677f44",
			"2022-5-15": "ami-0acc51c3357f26240",
			"2023":      "ami-0505d2a2a44257d17",
			"2":         "ami-00aae26e31bb072a2",
		},
	},
	"amazon-linux-ecs": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"2023": "ami-0307e11f511d976b0",
			"2":    "ami-0293ff221e87260aa",
		},
		"arm64": PlatformsAMIsType{
			"2023": "ami-0729b1e535f19c7b8",
			"2":    "ami-07af7b838076acdcc",
		},
	},
	"debian": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"9":  "ami-0182559468c1975fe",
			"10": "ami-0c0131c7dd91f82ea",
			"11": "ami-0136ba2af1041319b",
			"12": "ami-03b080214eb370bf2",
		},
		"arm64": PlatformsAMIsType{
			"10": "ami-054d2bc47c1082594",
			"11": "ami-089492faff470d87d",
			"12": "ami-07e7ff4fc4bfb342e",
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
	"fedora": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"40": "ami-0fac75d0443e36a79",
		},
		"arm64": PlatformsAMIsType{
			"42": "ami-0b2d98d9724ad62c3",
		},
	},
	"macos": PlatformsArchsType{
		"arm64": PlatformsAMIsType{
			"sonoma": "ami-0c582a76ed8159789",
		},
		"x86_64": PlatformsAMIsType{
			"sonoma": "ami-0af4746d79fd670cd",
		},
	},
	"redhat": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"9":       "ami-0c878dd49ca800252",
			"86":      "ami-00064b50696aa0436",
			"86-fips": "ami-0d0fb96b595c56e03",
		},
		"arm64": PlatformsAMIsType{
			"9":  "ami-089b86d2f4d27cd98",
			"86": "ami-0d4438fbccc652f68",
		},
	},
	"rocky-linux": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"92": "ami-08f362c39d03a4eb5",
		},
	},
	"suse": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"12":   "ami-0b0597153739840c4",
			"15-4": "ami-025731bf74aa12ed6",
		},
		"arm64": PlatformsAMIsType{
			"15-4": "ami-0fee86fe165a4b4c3",
		},
	},
	"ubuntu": PlatformsArchsType{
		"x86_64": PlatformsAMIsType{
			"14-04": "ami-013d633d3b6cdb22c",
			"16-04": "ami-0dd82fcb5978fb075",
			"18-04": "ami-081753330aeb8a334",
			"20-04": "ami-0f22bf0f32cb71fb0",
			"22-04": "ami-01b34a3247328d55e",
			"23-04": "ami-04909211b4197c028",
			"23-10": "ami-0949b45ef274e55a1",
			"24-04": "ami-07458119f8579729d",
		},
		"arm64": PlatformsAMIsType{
			"18-04":   "ami-055744c75048d8296",
			"20-04":   "ami-062505f473642c789",
			"20-04-2": "ami-023f1e40b096c3ebc",
			"21-04":   "ami-0aa5218db2b0ff1d9",
			"22-04":   "ami-02490af8f731890a0",
			"23-04":   "ami-0820bcaf37ee46ff4",
			"23-10":   "ami-0dea732dd5f1da0a8",
			"24-04":   "ami-08a72149658f1eeea",
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
