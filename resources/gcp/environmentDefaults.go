package gcp

const (
	agentSandboxEnv = "gcp/agent-sandbox"
	agentQaEnv      = "gcp/agent-qa"
)

type environmentDefault struct {
	gcp     gcpProvider
	ddInfra ddInfra
}

type gcpProvider struct {
	project string
}

type ddInfra struct {
}

func getEnvironmentDefault(envName string) environmentDefault {
	switch envName {
	case agentSandboxEnv:
		return agentSandboxDefault()
	case agentQaEnv:
		return agentQaDefault()
	default:
		panic("Unknown environment: " + envName)
	}
}

func agentSandboxDefault() environmentDefault {
	return environmentDefault{
		gcp: gcpProvider{
			project: "",
		},
		ddInfra: ddInfra{},
	}
}

func agentQaDefault() environmentDefault {
	return environmentDefault{
		gcp: gcpProvider{
			project: "",
		},
		ddInfra: ddInfra{},
	}
}
