package asg

import (
	"encoding/base64"
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	resourcesEc2 "github.com/DataDog/test-infra-definitions/resources/aws/ec2"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const NODE_COUNT = 30

func Run(ctx *pulumi.Context) error {
	env, err := aws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	paramName := fmt.Sprintf("amzn2-ami-hvm-%s-gp2", "arm64")
	ami, err := resourcesEc2.GetAMIFromSSM(env, fmt.Sprintf("/aws/service/ami-amazon-linux-latest/%s", paramName))
	if err != nil {
		return err
	}

	// Create a new security group
	provider := env.GetProvider(config.ProviderAWS)
	sg, err := ec2.NewSecurityGroup(ctx, env.Namer.ResourceName("asg-sg"), &ec2.SecurityGroupArgs{
		VpcId: pulumi.String(env.DefaultVPCID()),
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(22),
				ToPort:     pulumi.Int(22),
				CidrBlocks: pulumi.ToStringArray([]string{"0.0.0.0/0"}),
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
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
	launchTemplate, err := resourcesEc2.NewEC2LaunchTemplate(env, env.Namer.ResourceName("asg-lt"), &resourcesEc2.LaunchTemplateArgs{
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

	// Create Autoscaling Group
	asg, err := resourcesEc2.NewAutoscalingGroup(env, env.Namer.ResourceName("asg"), launchTemplate.ID(), launchTemplate.LatestVersion, NODE_COUNT, NODE_COUNT, NODE_COUNT)
	if err != nil {
		return err
	}

	ctx.Export("ssh-command", pulumi.All(asg.Name, env.DefaultPrivateKeyPath()).ApplyT(
		func(args []interface{}) string {
			asgName := args[0].(string)
			privateKeyPath := args[1].(string)
			return fmt.Sprintf("aws ec2 describe-instances --filters 'Name=tag:aws:autoscaling:groupName,Values=%s' --query 'Reservations[].Instances[].PublicDnsName' --output text | xargs -L1 -I {} ssh -i %s ec2-user@{}", asgName, privateKeyPath)
		},
	))

	return nil
}
