package asg

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	resgcp "github.com/DataDog/test-infra-definitions/resources/gcp"

	gcpcompute "github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type zoneSetup struct {
	zone        string
	networkName string
	subnetName  string
	nodeCount   int
}

var zones = []zoneSetup{
	{
		zone:        "us-central1-a",
		networkName: "NETWORK_NAME_PLACEHOLDER",
		subnetName:  "SUBNET_NAME_PLACEHOLDER",
		nodeCount:   2,
	},
	{
		zone:        "us-east1-b",
		networkName: "NETWORK_NAME_PLACEHOLDER",
		subnetName:  "SUBNET_NAME_PLACEHOLDER",
		nodeCount:   2,
	},
	{
		zone:        "us-west1-b",
		networkName: "NETWORK_NAME_PLACEHOLDER",
		subnetName:  "SUBNET_NAME_PLACEHOLDER",
		nodeCount:   2,
	},
	{
		zone:        "europe-west1-b",
		networkName: "NETWORK_NAME_PLACEHOLDER",
		subnetName:  "SUBNET_NAME_PLACEHOLDER",
		nodeCount:   2,
	},
}

// Run creates a set of GCP VM instances with public IPs and a startup script
// that runs the Datadog Agent and a k6 load test container, similar to the AWS ASG scenario.
// Resource identifiers (network/subnetwork) can be customized in code or via environment defaults.
func Run(ctx *pulumi.Context) error {
	// Create a GCP environment (uses configured project/region/zone and network defaults)
	env, err := resgcp.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// Basic settings — adjust as needed
	machineType := env.DefaultInstanceType()
	image := "projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts"

	provider := env.GetProvider(config.ProviderGCP)

	// Create a firewall per distinct network to allow SSH to instances tagged with "asg-ssh"
	createdFirewallNetworks := map[string]bool{}

	// Prepare SSH metadata (public key) and startup script
	sshPublicKey, err := utils.GetSSHPublicKey(env.DefaultPublicKeyPath())
	if err != nil {
		return err
	}

	// For each configured zone, create the requested number of VMs
	for _, setup := range zones {
		// Create firewall for this network once
		if !createdFirewallNetworks[setup.networkName] {
			fwName := env.Namer.ResourceName(fmt.Sprintf("asg-ssh-allow-%s", strings.ReplaceAll(setup.networkName, "-", "")))
			_, err = gcpcompute.NewFirewall(ctx, fwName, &gcpcompute.FirewallArgs{
				Network: pulumi.String(setup.networkName),
				Allows: gcpcompute.FirewallAllowArray{
					&gcpcompute.FirewallAllowArgs{
						Protocol: pulumi.String("tcp"),
						Ports:    pulumi.ToStringArray([]string{"22"}),
					},
				},
				Direction:    pulumi.String("INGRESS"),
				SourceRanges: pulumi.ToStringArray([]string{"0.0.0.0/0"}),
				TargetTags:   pulumi.ToStringArray([]string{"asg-ssh"}),
			}, pulumi.Provider(provider))
			if err != nil {
				return err
			}
			createdFirewallNetworks[setup.networkName] = true
		}

		for i := 0; i < setup.nodeCount; i++ {
			idx := i + 1
			name := env.Namer.ResourceName(fmt.Sprintf("asg-vm-%s-%02d", strings.ReplaceAll(setup.zone, "-", ""), idx))

			// Startup script runs Docker, Datadog Agent, and k6 load test
			// Use AgentAPIKey from the environment
			startupScript := env.AgentAPIKey().ApplyT(func(apiKey string) (string, error) {
				script := fmt.Sprintf(`#!/bin/bash -ex
exec > >(tee /var/log/startup-script.log|logger -t startup-script -s 2>/dev/console) 2>&1

apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io
systemctl start docker
systemctl enable docker
docker run -d --name datadog-agent \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /proc/:/host/proc/:ro \
  -v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
  -p 8125:8125/udp \
  -e DD_API_KEY='%s' \
  -e DD_SITE="datadoghq.com" \
  -e DD_TAGS="ali-test:1" \
  -e DD_DOGSTATSD_NON_LOCAL_TRAFFIC=true \
  -e DD_DOGSTATSD_METRICS_STATS_ENABLE=true \
  public.ecr.aws/datadog/agent:latest
docker run -d --name k6-loadtest \
  --network host \
  -e K6_STATSD_ENABLE_TAGS=true \
  -e K6_STATSD_ADDR="localhost:8125" \
  -e K6_DISCARD_RESPONSE_BODIES=false \
  -e REGISTRY_HOST="adel-reg.com" \
  -e REPOSITORY="agent" \
  -e IMAGE_TAG="7.70.0" \
  --memory 2g \
  alidatadog/k6-loadtest:registry run --out output-statsd /home/k6/script.js
`, apiKey)
				// Some distros require base64 for MetadataStartupScript; we keep raw via metadata key below.
				return script, nil
			}).(pulumi.StringOutput)

			// Compose instance definition
			// Allocate ephemeral public IP via AccessConfigs with empty NatIp
			_, err := gcpcompute.NewInstance(ctx, name, &gcpcompute.InstanceArgs{
				Name:        env.Namer.DisplayName(63, pulumi.String(name)),
				MachineType: pulumi.String(machineType),
				Zone:        pulumi.String(setup.zone),
				BootDisk: &gcpcompute.InstanceBootDiskArgs{
					InitializeParams: &gcpcompute.InstanceBootDiskInitializeParamsArgs{
						Image: pulumi.String(image),
						Size:  pulumi.Int(50),
					},
				},
				NetworkInterfaces: gcpcompute.InstanceNetworkInterfaceArray{
					&gcpcompute.InstanceNetworkInterfaceArgs{
						Network:    pulumi.String(setup.networkName),
						Subnetwork: pulumi.String(setup.subnetName),
						AccessConfigs: gcpcompute.InstanceNetworkInterfaceAccessConfigArray{
							&gcpcompute.InstanceNetworkInterfaceAccessConfigArgs{
								NatIp: pulumi.String(""),
							},
						},
					},
				},
				Metadata: pulumi.StringMap{
					"enable-oslogin": pulumi.String("false"),
					"ssh-keys":       pulumi.Sprintf("gce:%s", sshPublicKey),
					"startup-script": pulumi.StringOutput(startupScript),
				},
				ServiceAccount: &gcpcompute.InstanceServiceAccountArgs{
					Email: pulumi.String(env.DefaultVMServiceAccount()),
					Scopes: pulumi.StringArray{
						pulumi.String("cloud-platform"),
					},
				},
				Tags: pulumi.StringArray{
					pulumi.String("asg-ssh"),
				},
			}, pulumi.Provider(provider))
			if err != nil {
				return err
			}
		}

		// Export a zone-specific SSH helper command
		ctx.Export(fmt.Sprintf("ssh-command-%s", setup.zone), pulumi.Sprintf("gcloud compute instances list --filter='tags.items=asg-ssh AND zone:(%s)' --format='value(networkInterfaces[0].accessConfigs[0].natIp)' | xargs -L1 -I {} ssh -i %s gce@{}", setup.zone, env.DefaultPrivateKeyPath()))
	}

	// Export a general note
	ctx.Export("gcp-asg-note", pulumi.String("Instances created across configured zones with tag 'asg-ssh'. Use exported ssh-command-<zone> outputs."))

	return nil
}
