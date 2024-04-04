package jmxfetch

import (
	_ "embed"

	"github.com/DataDog/test-infra-definitions/components/docker"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed docker-compose.yaml
var dockerComposeContent string

var DockerComposeManifest = docker.ComposeInlineManifest{
	Name:    "jmxfetch-test",
	Content: pulumi.String(dockerComposeContent),
}
