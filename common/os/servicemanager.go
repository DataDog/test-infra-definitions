package os

type ServiceManager struct{ restartCmd []string }

func (s *ServiceManager) RestartAgentCmd() []string {
	return s.restartCmd
}
