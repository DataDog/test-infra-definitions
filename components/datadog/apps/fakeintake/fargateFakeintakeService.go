package fakeintake

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	ecsClient "github.com/DataDog/test-infra-definitions/resources/aws/ecs"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/fakeintake/fakeintakeparams"
	"github.com/cenkalti/backoff/v4"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	sleepInterval = 1 * time.Second
	maxRetries    = 120
	containerName = "fakeintake"
	port          = 80
	cpu           = "256"
	memory        = "1024"
)

type Instance struct {
	pulumi.ResourceState

	Host pulumi.StringOutput
	Name string
}

func NewECSFargateInstance(e aws.Environment, option ...fakeintakeparams.Option) (*Instance, error) {
	params, paramsErr := fakeintakeparams.NewParams(option...)
	if paramsErr != nil {
		return nil, paramsErr
	}

	namer := e.Namer.WithPrefix(params.Name)
	opts := []pulumi.ResourceOption{e.WithProviders(config.ProviderAWS, config.ProviderAWSX)}

	instance := &Instance{
		Name: params.Name,
	}
	if err := e.Ctx.RegisterComponentResource("dd:fakeintake", namer.ResourceName("grp"), instance, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(instance))

	var balancerArray classicECS.ServiceLoadBalancerArray
	var alb *lb.ApplicationLoadBalancer
	var err error
	if params.LoadBalancerEnabled {
		alb, err = lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
			Name:           e.CommonNamer.DisplayName(32, pulumi.String(params.Name)),
			SubnetIds:      e.RandomSubnets(),
			Internal:       pulumi.BoolPtr(!e.ECSServicePublicIP()),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			DefaultTargetGroup: &lb.TargetGroupArgs{
				Name:       e.CommonNamer.DisplayName(32, pulumi.String(params.Name)),
				Port:       pulumi.IntPtr(port),
				Protocol:   pulumi.StringPtr("HTTP"),
				TargetType: pulumi.StringPtr("ip"),
				VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
			},
			Listener: &lb.ListenerArgs{
				Port:     pulumi.IntPtr(port),
				Protocol: pulumi.StringPtr("HTTP"),
			},
		}, opts...)
		if err != nil {
			return nil, err
		}

		instance.Host = alb.LoadBalancer.DnsName()
		balancerArray = classicECS.ServiceLoadBalancerArray{
			&classicECS.ServiceLoadBalancerArgs{
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(port),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		}
	} else {
		instance.Host, err = fargateServiceFakeintakeWithoutLoadBalancer(e, params.Name, params.ImageURL)
		if err != nil {
			return nil, err
		}
		balancerArray = classicECS.ServiceLoadBalancerArray{}
	}

	if _, err := ecs.NewFargateService(e.Ctx, namer.ResourceName("srv"), &ecs.FargateServiceArgs{
		Cluster:              pulumi.StringPtr(e.ECSFargateFakeintakeClusterArn()),
		Name:                 e.CommonNamer.DisplayName(255, pulumi.String(params.Name)),
		DesiredCount:         pulumi.IntPtr(1),
		EnableExecuteCommand: pulumi.BoolPtr(true),
		NetworkConfiguration: &classicECS.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.BoolPtr(false),
			SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
			Subnets:        e.RandomSubnets(),
		},
		LoadBalancers: balancerArray,
		TaskDefinitionArgs: &ecs.FargateServiceTaskDefinitionArgs{
			Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
				containerName: *fargateLinuxContainerDefinition(params.ImageURL),
			},
			Cpu:    pulumi.StringPtr(cpu),
			Memory: pulumi.StringPtr(memory),
			ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
			},
			TaskRole: &awsx.DefaultRoleWithPolicyArgs{
				RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
			},
			Family: e.CommonNamer.DisplayName(255, pulumi.String("fakeintake-ecs")),
		},
		ContinueBeforeSteadyState: pulumi.BoolPtr(true),
	}, opts...); err != nil {
		return nil, err
	}
	if err := e.Ctx.RegisterResourceOutputs(instance, pulumi.Map{
		"host": instance.Host,
	}); err != nil {
		return nil, err
	}

	return instance, nil
}

func fargateLinuxTaskDefinition(e aws.Environment, name, imageURL string) (*ecs.FargateTaskDefinition, error) {
	return ecs.NewFargateTaskDefinition(e.Ctx, e.Namer.ResourceName(name), &ecs.FargateTaskDefinitionArgs{
		Containers: map[string]ecs.TaskDefinitionContainerDefinitionArgs{
			containerName: *fargateLinuxContainerDefinition(imageURL),
		},
		Cpu:    pulumi.StringPtr(cpu),
		Memory: pulumi.StringPtr(memory),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskExecutionRole()),
		},
		TaskRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: pulumi.StringPtr(e.ECSTaskRole()),
		},
		Family: e.CommonNamer.DisplayName(13, pulumi.String("fakeintake-ecs")),
	}, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))
}

func fargateLinuxContainerDefinition(imageURL string) *ecs.TaskDefinitionContainerDefinitionArgs {
	return &ecs.TaskDefinitionContainerDefinitionArgs{
		Name:        pulumi.String(containerName),
		Image:       pulumi.String(imageURL),
		Essential:   pulumi.BoolPtr(true),
		MountPoints: ecs.TaskDefinitionMountPointArray{},
		Environment: ecs.TaskDefinitionKeyValuePairArray{
			ecs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("GOMEMLIMIT"),
				Value: pulumi.StringPtr("768MiB"),
			},
		},
		PortMappings: ecs.TaskDefinitionPortMappingArray{
			ecs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(port),
				HostPort:      pulumi.Int(port),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		VolumesFrom: ecs.TaskDefinitionVolumeFromArray{},
		HealthCheck: &ecs.TaskDefinitionHealthCheckArgs{
			// note that a failing health check doesn't fail the deployment,
			// but it allows seeing the health of the task directly in AWS
			Command: pulumi.StringArray{
				pulumi.String("curl"),
				pulumi.String("-L"),
				pulumi.String(getFakeintakeHealthURL("localhost")),
			},
			Interval: pulumi.Int(5), // seconds
			Retries:  pulumi.Int(4),
		},
	}
}

// fargateServiceFakeintakeWithoutLoadBalancer deploys one fakeintake container to a dedicated Fargate cluster
// Hardcoded on sandbox
func fargateServiceFakeintakeWithoutLoadBalancer(e aws.Environment, name, imageURL string) (ipAddress pulumi.StringOutput, err error) {
	taskDef, err := fargateLinuxTaskDefinition(e, e.Namer.ResourceName(name, "taskdef"), imageURL)
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	fargateService, err := ecsClient.FargateService(e, e.Namer.ResourceName(name, "srv"), pulumi.String(e.ECSFargateFakeintakeClusterArn()), taskDef.TaskDefinition.Arn())
	// Hack passing taskDef.TaskDefinition.Arn() to execute apply function
	// when taskDef has an ARN, thus it is defined on AWS side
	ipAddress = pulumi.All(taskDef.TaskDefinition.Arn(), fargateService.Service.Name()).ApplyT(func(args []any) (string, error) {
		var ipAddress string
		err := backoff.Retry(func() error {
			e.Ctx.Log.Debug("waiting for fakeintake task private ip", nil)
			serviceName := args[1].(string)
			ecsClient, err := ecsClient.NewECSClient(e.Ctx.Context(), e.Region())
			if err != nil {
				return err
			}
			ipAddress, err = ecsClient.GetTaskPrivateIP(e.ECSFargateFakeintakeClusterArn(), serviceName)
			if err != nil {
				return err
			}
			e.Ctx.Log.Info(fmt.Sprintf("fakeintake task private ip found: %s\n", ipAddress), nil)
			return err
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(sleepInterval), maxRetries))

		if err != nil {
			return "", err
		}

		// fail the deployment if the fakeintake is not healthy
		e.Ctx.Log.Info(fmt.Sprintf("Waiting for fakeintake at %s to be healthy", ipAddress), nil)
		err = backoff.Retry(func() error {
			url := getFakeintakeHealthURL(ipAddress)
			e.Ctx.Log.Debug(fmt.Sprintf("getting fakeintake health at %s", url), nil)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("error getting fakeintake health: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			}
			return nil
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(sleepInterval), maxRetries))

		return ipAddress, err
	}).(pulumi.StringOutput)

	return ipAddress, err
}

func getFakeintakeHealthURL(host string) string {
	url := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/fakeintake/health",
	}
	return url.String()
}
