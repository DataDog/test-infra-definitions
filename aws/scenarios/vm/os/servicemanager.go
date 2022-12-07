package os

type serviceManager struct{ startCmd string }

func (s *serviceManager) StartAgentCmd() string {
	return s.startCmd
}
