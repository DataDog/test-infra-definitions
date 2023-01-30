package utils

import "fmt"

func BuildDockerImagePath(dockerRepository string, imageVersion string) string {
	return fmt.Sprintf("%s:%s", dockerRepository, imageVersion)
}
