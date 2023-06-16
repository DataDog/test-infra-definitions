package apmautoinjector

import (
	"github.com/DataDog/test-infra-definitions/common"
)

// Params defines the parameters for the APM auto-injector installation.
// The Params configuration uses the [Functional options pattern].
//
// The available options are:
//   - [WithLocalInstallerPath]
//   - [WithInstallArgs]
//
// [Functional options pattern]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Params struct {
	localInstallerPath string
	installArgs        string
}

func newParams(options ...func(*Params) error) (*Params, error) {
	p := &Params{}
	return common.ApplyOption(p, options)
}

func WithLocalInstallerPath(path string) func(*Params) error {
	return func(p *Params) error {
		p.localInstallerPath = path
		return nil
	}
}

func WithInstallArgs(args string) func(*Params) error {
	return func(p *Params) error {
		p.installArgs = args
		return nil
	}
}
