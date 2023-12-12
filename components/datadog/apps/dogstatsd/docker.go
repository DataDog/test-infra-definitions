package dogstatsd

import (
	_ "embed"
)

//go:embed docker-compose.yaml
var DockerComposeDefinition string
