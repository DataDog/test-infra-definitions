package compute

import (
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/DataDog/test-infra-definitions/components/os"
)

type vmArgs struct {
	osInfo       *os.Descriptor
	instanceType string
	imageName    string
}

type VMOption = func(*vmArgs) error

func newParams(options ...VMOption) (*vmArgs, error) {
	vmArgs := &vmArgs{}

	return common.ApplyOption(vmArgs, options)
}
