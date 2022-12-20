package os

type serviceManager struct{ restartCmd []string }

func (s *serviceManager) RestartAgentCmd() []string {
	return s.restartCmd
}
