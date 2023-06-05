package vm

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/components/command"
	commonos "github.com/DataDog/test-infra-definitions/components/os"
)

type UnixVM struct {
	packageManager    command.PackageManager
	runner            *command.Runner
	lazyDockerManager *command.DockerManager
	VM
}

func NewUnixVM(vm VM) (*UnixVM, error) {
	os := vm.GetOS()
	if os.GetType() == commonos.WindowsType {
		return nil, errors.New("the OS Windows is not a valid Nix OS. Use `NewXXXVM` instead of `NewNixXXXVM`")
	}

	runner := vm.GetRunner()
	packageManager, err := os.CreatePackageManager(runner)
	if err != nil {
		return nil, err
	}
	return &UnixVM{
		VM:             vm,
		runner:         runner,
		packageManager: packageManager,
	}, nil

}
func (vm *UnixVM) GetPackageManager() command.PackageManager {
	return vm.packageManager
}

func (vm *UnixVM) GetLazyDocker() *command.DockerManager {
	if vm.lazyDockerManager == nil {
		vm.lazyDockerManager = command.NewDockerManager(vm.GetRunner(), vm.GetPackageManager())
	}
	return vm.lazyDockerManager
}
