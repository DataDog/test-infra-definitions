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
	region  string
}

type ddInfra struct {
	defaultInstanceType string
	defaultNetworkName  string
	defaultSubnetName   string
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
			project: "datadog-agent-sandbox",
			region:  "us-central1-a",
		},
		ddInfra: ddInfra{
			defaultInstanceType: "e2-medium",
			defaultNetworkName:  "datadog-agent-sandbox-us-central1",
			defaultSubnetName:   "datadog-agent-sandbox-us-central1-private",
		},
	}
}

func agentQaDefault() environmentDefault {
	return environmentDefault{
		gcp: gcpProvider{
			project: "datadog-agent-qa",
			region:  "us-central1-a",
		},
		ddInfra: ddInfra{
			defaultInstanceType: "e2-medium",
			defaultNetworkName:  "datadog-agent-qa-us-central1",
			defaultSubnetName:   "datadog-agent-qa-us-central1-private",
		},
	}
}
