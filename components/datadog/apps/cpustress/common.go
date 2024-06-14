package cpustress

import "fmt"

const (
	defaultStressNgImageRepo = "ghcr.io/colinianking/stress-ng"
	defaultStressNgImageTag  = "9e9d6045b5e938f279f7e802fb72cfcf0eb261f9"
)

func getStressNGImage() string {
	return fmt.Sprintf("%s:%s", defaultStressNgImageRepo, defaultStressNgImageTag)
}
