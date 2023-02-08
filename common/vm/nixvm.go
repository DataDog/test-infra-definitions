package vm

import (
	"errors"

	"github.com/DataDog/test-infra-definitions/command"
	commonos "github.com/DataDog/test-infra-definitions/common/os"
)

type NixVM struct {
	aptManager        *command.AptManager
	fileManager       *command.FileManager
	runner            *command.Runner
	lazyDockerManager *command.DockerManager
	VM
}

func NewNixVM(vm VM) (*NixVM, error) {
	if vm.GetOS().GetType() == commonos.WindowsType {
		return nil, errors.New("the OS Windows is not a valid Nix OS. Use `NewXXXVM` instead of `NewNixXXXVM`")
	}
	runner := vm.GetRunner()
	return &NixVM{
		VM:          vm,
		runner:      runner,
		aptManager:  command.NewAptManager(runner),
		fileManager: command.NewFileManager(runner),
	}, nil

}
func (vm *NixVM) GetAptManager() *command.AptManager {
	return vm.aptManager
}

func (vm *NixVM) GetFileManager() *command.FileManager {
	return vm.fileManager
}

func (vm *NixVM) GetLazyDocker() *command.DockerManager {
	if vm.lazyDockerManager == nil {
		vm.lazyDockerManager = command.NewDockerManager(vm.GetRunner(), vm.GetAptManager())
	}
	return vm.lazyDockerManager
}
