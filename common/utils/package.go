package utils

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	tifos "github.com/DataDog/test-infra-definitions/components/os"
)

// GetPackagePath retrieve the name of the package that should be installed.
// It will return the path to the package that should be installed for the given flavor and agent flavor.
// If the package is not found, it will return an error.
// If multiple packages are found, it will return the first one and print a warning.
func GetPackagePath(localPath string, flavor tifos.Flavor, agentFlavor string) (string, error) {
	var wantedExt string
	switch flavor {
	case tifos.AmazonLinux, tifos.CentOS, tifos.RedHat, tifos.AmazonLinuxECS, tifos.Fedora, tifos.Suse, tifos.RockyLinux:
		wantedExt = ".rpm"
	case tifos.Debian, tifos.Ubuntu:
		wantedExt = ".deb"
	case tifos.WindowsServer:
		wantedExt = ".msi"
	case tifos.MacosOS, tifos.Unknown:
		fallthrough
	default:
		return "", fmt.Errorf("unsupported flavor for local packages installation: %s", flavor)
	}

	pathInfo, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	packagePath := localPath
	matches := []string{}
	if pathInfo.IsDir() {
		entries, err := os.ReadDir(localPath)
		if err != nil {
			return "", err
		}

		// First match all packages with the correct extension
		allPackagesPattern := `.*\.` + strings.TrimPrefix(wantedExt, ".") + `$`
		fipsPattern := `.*fips.*\.` + strings.TrimPrefix(wantedExt, ".") + `$`

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// It would have been easier to use ^(?!.*fips).*\.deb$ with lookahead to match non-FIPS packages, but it is not supported by Go regex.
			// Instead we get all the packages and filter out the FIPS ones if we're looking for non-FIPS packages.
			if matched, _ := regexp.MatchString(allPackagesPattern, entry.Name()); matched {
				// If we're looking for FIPS packages, only include those
				if agentFlavor == agentparams.FIPSFlavor {
					if matched, _ := regexp.MatchString(fipsPattern, entry.Name()); matched {
						matches = append(matches, entry.Name())
					}
				} else {
					// If we're looking for non-FIPS packages, exclude FIPS ones
					if matched, _ := regexp.MatchString(fipsPattern, entry.Name()); !matched {
						matches = append(matches, entry.Name())
					}
				}
			}
		}

		if len(matches) == 0 {
			if agentFlavor == agentparams.FIPSFlavor {
				return "", fmt.Errorf("no FIPS package found in %s matching pattern %s", localPath, fipsPattern)
			}
			return "", fmt.Errorf("no package found in %s matching pattern %s without matching FIPS pattern %s", localPath, allPackagesPattern, fipsPattern)
		}

		if len(matches) > 1 {
			fmt.Printf("Found multiple packages to install, using the first one: %s", matches[0])
		}
		packagePath = path.Join(packagePath, matches[0])
	} else {
		if strings.HasSuffix(localPath, wantedExt) {
			matches = append(matches, path.Base(localPath))
		} else {
			return "", fmt.Errorf("local package %s does not have the expected extension %s", localPath, wantedExt)
		}
	}
	return packagePath, nil
}
