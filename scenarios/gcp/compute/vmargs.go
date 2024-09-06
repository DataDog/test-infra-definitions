package compute

import "github.com/DataDog/test-infra-definitions/components/os"

type vmArgs struct {
	osInfo       *os.Descriptor
	instanceType string
	imageName    string
}

type VMOption func(*vmArgs) error

func newParams(options ...VMOption) (*vmArgs, error) {
	return &vmArgs{}, nil
}
