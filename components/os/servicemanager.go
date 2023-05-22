package os

type ServiceManager struct{ restartCmd []string }

func (s *ServiceManager) RestartAgentCmd() []string {
	return s.restartCmd
}

func NewSystemCtlServiceManager() *ServiceManager {
	return &ServiceManager{
		restartCmd: []string{"sudo systemctl restart datadog-agent"},
	}
}

func NewServiceCmdServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{"sudo service datadog-agent restart"}}
}
