package redis

import (
	_ "embed"

	"github.com/DataDog/test-infra-definitions/components/docker"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed docker-compose.yaml
var dockerComposeYAML string

var DockerComposeManifest = docker.ComposeInlineManifest{
	Name:    "redis",
	Content: pulumi.String(dockerComposeYAML),
}
