package utils

import (
	"fmt"
	"strings"
)

func BuildDockerImagePath(dockerRepository string, imageVersion string) string {
	return fmt.Sprintf("%s:%s", dockerRepository, imageVersion)
}

func ParseImageReference(imageRef string) (imagePath string, tag string) {
	tagSepIdx := strings.LastIndex(imageRef, ":")
	if tagSepIdx == -1 {
		// no tag, tag is latest
		imagePath = imageRef
		tag = "latest"
		return
	}

	imagePath = imageRef[0:tagSepIdx]
	tag = imageRef[tagSepIdx+1:]
	return
}
