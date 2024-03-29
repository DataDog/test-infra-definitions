package cpustress

import "fmt"

const (
	defaultStressNgImageRepo = "ghcr.io/colinianking/stress-ng"
	defaultStressNgImageTag  = "409201de7458c639c68088d28ec8270ef599fe47"
)

func getStressNGImage() string {
	return fmt.Sprintf("%s:%s", defaultStressNgImageRepo, defaultStressNgImageTag)
}
