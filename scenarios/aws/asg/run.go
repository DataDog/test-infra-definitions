package asg

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	resourcesEc2 "github.com/DataDog/test-infra-definitions/resources/aws/ec2"
	"github.com/samber/lo"

	awsec2 "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type regionSetup struct {
	region    string
	vpcID     string
	subnetIDs []string
	nodeCount int
}

var regions = []regionSetup{
	{
		region:    "us-east-1",
		vpcID:     "vpc-029c0faf8f49dee8d",
		subnetIDs: []string{"subnet-0f161570a33bc6ca9", "subnet-039ef29c53fff2c90", "subnet-06073afc873f61b1a"},
		nodeCount: 2,
	},
	{
		region:    "us-east-2",
		vpcID:     "vpc-0099125df157e538f",
		subnetIDs: []string{"subnet-09e84e5e4ef266c5c", "subnet-0fce0f18de77c4d26", "subnet-052fb34effb5c7fc3"},
		nodeCount: 2,
	},
	{
		region:    "us-west-1",
		vpcID:     "vpc-090398e60baa5bda0",
		subnetIDs: []string{"subnet-0d9c8d4d8f8f6cd1a", "subnet-04daf8136604ff2c2"},
		nodeCount: 2,
	},
	{
		region:    "us-west-2",
		vpcID:     "vpc-09ff8901d274f393c",
		subnetIDs: []string{"subnet-0b83fd99ba97eb56c", "subnet-024d7efd08e381a5f", "subnet-0454d4998fbdc3cc8", "subnet-060a5177b1597a8b8"},
		nodeCount: 2,
	},
	{
		region:    "eu-central-1",
		vpcID:     "vpc-02f1ebfd05a562247",
		subnetIDs: []string{"subnet-0d829649634c6cb32", "subnet-027b9575ef024d8b6", "subnet-05b4b29a11b53db2b"},
		nodeCount: 2,
	},
}

func Run(ctx *pulumi.Context) error {
	// Create a shared CommonEnvironment to avoid provider duplication
	commonEnv, err := config.NewCommonEnvironment(ctx)
	if err != nil {
		return err
	}

	for _, setup := range regions {
		// Create a per-region environment overriding region; network will be taken from config defaults for that env
		env, err := aws.NewEnvironment(ctx, aws.WithCommonEnvironment(&commonEnv), aws.WithPrefix(setup.region), aws.WithRegion(setup.region), aws.WithNetwork(setup.vpcID, setup.subnetIDs))
		if err != nil {
			return err
		}

		// Resolve a region-specific Amazon Linux 2 AMI via SSM
		paramName := fmt.Sprintf("amzn2-ami-hvm-%s-gp2", "arm64")
		ami, err := resourcesEc2.GetAMIFromSSM(env, fmt.Sprintf("/aws/service/ami-amazon-linux-latest/%s", paramName))
		if err != nil {
			return err
		}

		// Create a region-scoped security group in the default VPC
		provider := env.GetProvider(config.ProviderAWS)
		sgName := env.Namer.ResourceName(fmt.Sprintf("asg-sg-%s", strings.ReplaceAll(setup.region, "-", "")))
		sg, err := awsec2.NewSecurityGroup(ctx, sgName, &awsec2.SecurityGroupArgs{
			VpcId: pulumi.String(env.DefaultVPCID()),
			Ingress: awsec2.SecurityGroupIngressArray{
				awsec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.ToStringArray([]string{"0.0.0.0/0"}),
				},
			},
			Egress: awsec2.SecurityGroupEgressArray{
				awsec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.ToStringArray([]string{"0.0.0.0/0"}),
				},
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}

		// Create Launch Template
		ltName := env.Namer.ResourceName(fmt.Sprintf("asg-lt-%s", strings.ReplaceAll(setup.region, "-", "")))
		launchTemplate, err := resourcesEc2.NewEC2LaunchTemplate(env, ltName, &resourcesEc2.LaunchTemplateArgs{
			InstanceType:     pulumi.String("t4g.small"),
			ImageID:          pulumi.String(ami),
			KeyName:          pulumi.String(env.DefaultKeyPairName()),
			SecurityGroupIDs: pulumi.StringArray{sg.ID()},
			UserData: env.AgentAPIKey().ApplyT(func(apiKey string) (string, error) {
				userData := fmt.Sprintf(`#!/bin/bash -ex
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

sudo yum update -y
sudo amazon-linux-extras install docker -y
sudo systemctl start docker
sudo systemctl enable docker
sudo docker run -d --name datadog-agent \
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
sudo docker run -d --name k6-loadtest \
  --network host \
  -e K6_STATSD_ENABLE_TAGS=true \
  -e K6_STATSD_ADDR="localhost:8125" \
  -e K6_DISCARD_RESPONSE_BODIES=true \
  --memory 2g \
  alidatadog/k6-loadtest:s3 run --out output-statsd /home/k6/script.js
`, apiKey)
				return base64.StdEncoding.EncodeToString([]byte(userData)), nil
			}).(pulumi.StringInput),
			AssociatePublicIp: true,
		})
		if err != nil {
			return err
		}

		// Create Autoscaling Group per region
		asgName := env.Namer.ResourceName(fmt.Sprintf("asg-%s", strings.ReplaceAll(setup.region, "-", "")))
		asg, err := resourcesEc2.NewAutoscalingGroup(env, asgName, launchTemplate.ID(), launchTemplate.LatestVersion, setup.nodeCount, setup.nodeCount, setup.nodeCount)
		if err != nil {
			return err
		}

		// Export a region-specific SSH command helper
		ctx.Export(fmt.Sprintf("ssh-command-%s", setup.region), pulumi.All(asg.Name, env.DefaultPrivateKeyPath()).ApplyT(
			func(args []interface{}) string {
				asgName := args[0].(string)
				privateKeyPath := args[1].(string)
				return fmt.Sprintf("aws ec2 describe-instances --region %s --filters 'Name=tag:aws:autoscaling:groupName,Values=%s' --query 'Reservations[].Instances[].PublicDnsName' --output text | xargs -L1 -I {} ssh -i %s ec2-user@{}", setup.region, asgName, privateKeyPath)
			},
		))
	}

	// Also export the regions used
	ctx.Export("regions", pulumi.ToStringArray(lo.Map(regions, func(setup regionSetup, _ int) string {
		return setup.region
	})))
	return nil
}
