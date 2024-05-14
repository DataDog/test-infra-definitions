package docker

const (
	sandboxEnv = "dclocal/sandbox"
)

type environmentDefault struct {
	ddInfra ddInfra
}

type ddInfra struct{}

func getEnvironmentDefault(envName string) environmentDefault {
	switch envName {
	case sandboxEnv:
		return sandboxDefault()
	default:
		panic("Unknown environment: " + envName)
	}
}

func sandboxDefault() environmentDefault {
	return environmentDefault{}
}
