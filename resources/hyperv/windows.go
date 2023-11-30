package hyperv

import (
	"github.com/DataDog/test-infra-definitions/components/os"
)

type windows struct {
	*os.Windows
}

func newWindows(env Environment) *windows {
	return &windows{
		Windows: os.NewWindows(&env),
	}
}

func (*windows) GetSSHUser() string { return "<YOUR_VM_USERNAME>" }

func (w *windows) GetImage(os.Architecture) (string, error) {
	return "", nil
}
