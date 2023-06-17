package driver

import (
	"github.com/DataDog/test-infra-definitions/common"
)

// Params defines the parameters for the Driver installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithLocalAssetDir]
//   - [WithInstallerName]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Params struct {
	localAssetDir string
	installerName string
}

func newParams(options ...func(*Params) error) (*Params, error) {
	p := &Params{
		localAssetDir: "",
		installerName: "datadog-apm-inject.msi",
	}
	return common.ApplyOption(p, options)
}

func WithLocalAssetDir(dir string) func(*Params) error {
	return func(p *Params) error {
		p.localAssetDir = dir
		return nil
	}
}

func WithInstallerName(name string) func(*Params) error {
	return func(p *Params) error {
		p.installerName = name
		return nil
	}
}
