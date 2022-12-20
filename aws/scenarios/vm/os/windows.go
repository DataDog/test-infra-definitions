package os

import (
	"github.com/DataDog/test-infra-definitions/aws"
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
)

type windows struct {
	env aws.Environment
}

func (*windows) GetSSHUser() string { panic("Not Yet supported") }

func (w *windows) GetAMI(arch Architecture) (string, error) {
	return ec2.SearchAMI(w.env, "801119661308", "Windows_Server-2022-English-Full-Base-*", string(arch))
}

func (*windows) GetAMIArch(arch Architecture) string { return string(arch) }
func (w *windows) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(w.env, arch)
}

func (*windows) GetTenancy() string { return "default" }

func (*windows) GetServiceManager() *serviceManager {
	return &serviceManager{restartCmd: []string{`%ProgramFiles%\Datadog\Datadog Agent\bin\agent.exe restart-service`}}
}

func (*windows) GetConfigPath() string { return `C:\ProgramData\Datadog\datadog.yaml` }

func (*windows) GetOSType() OSType { return WindowsOS }
