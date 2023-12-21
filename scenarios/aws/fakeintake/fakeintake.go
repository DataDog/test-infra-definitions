package fakeintake

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/resources/aws/ecs"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acm"
	calb "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/alb"
	classicECS "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	clb "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ssm"
	awsxEcs "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/lb"
	"github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/cenkalti/backoff/v4"
)

const (
	sleepInterval = 1 * time.Second
	maxRetries    = 120
	containerName = "fakeintake"
	port          = 80
	sslPort       = 443
)

func NewECSFargateInstance(e aws.Environment, name string, option ...Option) (*fakeintake.Fakeintake, error) {
	params, paramsErr := NewParams(option...)
	if paramsErr != nil {
		return nil, paramsErr
	}

	return components.NewComponent(*e.CommonEnvironment, e.Namer.ResourceName(name), func(fi *fakeintake.Fakeintake) error {
		namer := e.Namer.WithPrefix("fakeintake").WithPrefix(name)
		opts := []pulumi.ResourceOption{pulumi.Parent(fi)}

		apiKeyParam, err := ssm.NewParameter(e.Ctx, namer.ResourceName("agent", "apikey"), &ssm.ParameterArgs{
			Name:  e.CommonNamer.DisplayName(1011, pulumi.String(name), pulumi.String("apikey")),
			Type:  ssm.ParameterTypeSecureString,
			Value: e.AgentAPIKey(),
		}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS))...)
		if err != nil {
			return err
		}

		taskDef, err := ecs.FargateTaskDefinitionWithAgent(e,
			namer.ResourceName("taskdef"),
			pulumi.String("fakeintake-ecs"),
			1024, 4096,
			map[string]awsxEcs.TaskDefinitionContainerDefinitionArgs{"fakeintake": *fargateLinuxContainerDefinition(params.ImageURL, apiKeyParam.Name)},
			apiKeyParam.Name,
			nil,
			opts...,
		)
		if err != nil {
			return err
		}

		if params.LoadBalancerEnabled {
			err = fargateSvcLB(e, namer, taskDef, fi, opts...)
		} else {
			err = fargateSvcNoLB(e, namer, taskDef, fi, opts...)
		}
		if err != nil {
			return err
		}

		return nil
	})
}

// fargateSvcNoLB deploys one fakeintake container to a dedicated Fargate cluster
// Hardcoded on sandbox
func fargateSvcNoLB(e aws.Environment, namer namer.Namer, taskDef *awsxEcs.FargateTaskDefinition, fi *fakeintake.Fakeintake, opts ...pulumi.ResourceOption) error {
	fargateService, err := ecs.FargateService(e, namer.ResourceName("srv"), pulumi.String(e.ECSFargateFakeintakeClusterArn()), taskDef.TaskDefinition.Arn(), nil, opts...)
	if err != nil {
		return err
	}

	// Hack passing taskDef.TaskDefinition.Arn() to execute apply function
	// when taskDef has an ARN, thus it is defined on AWS side
	output := pulumi.All(taskDef.TaskDefinition.Arn(), fargateService.Service.Name()).ApplyT(func(args []any) ([]string, error) {
		serviceName := args[1].(string)
		var ipAddress string
		err := backoff.Retry(func() error {
			e.Ctx.Log.Debug("waiting for fakeintake task private ip", nil)
			ecsClient, err := ecs.NewECSClient(e.Ctx.Context(), e.Region())
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
			return nil, err
		}

		// fail the deployment if the fakeintake is not healthy
		e.Ctx.Log.Info(fmt.Sprintf("Waiting for fakeintake at %s to be healthy", ipAddress), nil)
		healthURL := buildFakeIntakeURL("http", ipAddress, "/fakeintake/health", port)
		err = backoff.Retry(func() error {
			e.Ctx.Log.Debug(fmt.Sprintf("getting fakeintake health at %s", healthURL), nil)
			resp, err := http.Get(healthURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("error getting fakeintake health: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			}
			return nil
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(sleepInterval), maxRetries))
		if err != nil {
			return nil, err
		}

		return []string{ipAddress, buildFakeIntakeURL("http", ipAddress, "", port)}, nil
	}).(pulumi.StringArrayOutput)

	fi.Host = output.Index(pulumi.Int(0))
	fi.URL = output.Index(pulumi.Int(1))
	fi.Scheme = pulumi.String("http").ToStringOutput()

	return err
}

func fargateSvcLB(e aws.Environment, namer namer.Namer, taskDef *awsxEcs.FargateTaskDefinition, fi *fakeintake.Fakeintake, opts ...pulumi.ResourceOption) error {
	alb, err := lb.NewApplicationLoadBalancer(e.Ctx, namer.ResourceName("lb"), &lb.ApplicationLoadBalancerArgs{
		Name:           e.CommonNamer.DisplayName(32, pulumi.String(fi.Name())),
		SubnetIds:      e.RandomSubnets(),
		Internal:       pulumi.BoolPtr(!e.ECSServicePublicIP()),
		SecurityGroups: pulumi.ToStringArray(e.DefaultSecurityGroups()),
		DefaultTargetGroup: &lb.TargetGroupArgs{
			HealthCheck: &clb.TargetGroupHealthCheckArgs{
				Path: pulumi.StringPtr("/fakeintake/health"),
			},
			Name:       e.CommonNamer.DisplayName(32, pulumi.String("name")),
			Port:       pulumi.IntPtr(port),
			Protocol:   pulumi.StringPtr("HTTP"),
			TargetType: pulumi.StringPtr("ip"),
			VpcId:      pulumi.StringPtr(e.DefaultVPCID()),
		},
		Listener: &lb.ListenerArgs{
			Port:     pulumi.IntPtr(port),
			Protocol: pulumi.StringPtr("HTTP"),
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS, config.ProviderAWSX))...)
	if err != nil {
		return err
	}

	key, err := tls.NewPrivateKey(e.Ctx, namer.ResourceName("key"), &tls.PrivateKeyArgs{
		Algorithm: pulumi.String("RSA"),
		RsaBits:   pulumi.IntPtr(4096),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderTLS))...)
	if err != nil {
		return err
	}

	selfcert, err := tls.NewSelfSignedCert(e.Ctx, namer.ResourceName("cert"), &tls.SelfSignedCertArgs{
		AllowedUses: pulumi.StringArray{
			pulumi.String("server_auth"),
		},
		DnsNames: pulumi.StringArray{
			alb.LoadBalancer.DnsName(),
		},
		PrivateKeyPem: key.PrivateKeyPem,
		Subject: &tls.SelfSignedCertSubjectArgs{
			CommonName: alb.LoadBalancer.DnsName(),
		},
		ValidityPeriodHours: pulumi.Int(24),
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderTLS))...)
	if err != nil {
		return err
	}

	cert, err := acm.NewCertificate(e.Ctx, namer.ResourceName("cert"), &acm.CertificateArgs{
		CertificateBody: selfcert.CertPem,
		PrivateKey:      key.PrivateKeyPem,
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS))...)
	if err != nil {
		return err
	}

	if _, err = calb.NewListener(e.Ctx, namer.ResourceName("lb-https"), &calb.ListenerArgs{
		LoadBalancerArn: alb.LoadBalancer.Arn(),
		Port:            pulumi.IntPtr(sslPort),
		Protocol:        pulumi.StringPtr("HTTPS"),
		CertificateArn:  cert.Arn,
		DefaultActions: calb.ListenerDefaultActionArray{
			calb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: alb.DefaultTargetGroup.Arn(),
			},
		},
	}, utils.MergeOptions(opts, e.WithProviders(config.ProviderAWS))...); err != nil {
		return err
	}
	ipAdress := alb.LoadBalancer.DnsName()
	balancerArray := classicECS.ServiceLoadBalancerArray{
		&classicECS.ServiceLoadBalancerArgs{
			ContainerName:  pulumi.String(containerName),
			ContainerPort:  pulumi.Int(port),
			TargetGroupArn: alb.DefaultTargetGroup.Arn(),
		},
	}

	_, err = ecs.FargateService(e, namer.ResourceName("srv"), pulumi.String(e.ECSFargateFakeintakeClusterArn()), taskDef.TaskDefinition.Arn(), balancerArray, opts...)
	if err != nil {
		return err
	}

	fi.Host = ipAdress
	fi.Scheme = pulumi.String("http").ToStringOutput()
	fi.URL = ipAdress.ApplyT(func(ip string) string {
		return buildFakeIntakeURL("http", ip, "", port)
	}).(pulumi.StringOutput)

	return nil
}

func fargateLinuxContainerDefinition(imageURL string, apiKeySSMParamName pulumi.StringInput) *awsxEcs.TaskDefinitionContainerDefinitionArgs {
	return &awsxEcs.TaskDefinitionContainerDefinitionArgs{
		Name:        pulumi.String(containerName),
		Image:       pulumi.String(imageURL),
		Essential:   pulumi.BoolPtr(true),
		MountPoints: awsxEcs.TaskDefinitionMountPointArray{},
		Environment: awsxEcs.TaskDefinitionKeyValuePairArray{
			awsxEcs.TaskDefinitionKeyValuePairArgs{
				Name:  pulumi.StringPtr("GOMEMLIMIT"),
				Value: pulumi.StringPtr("3072MiB"),
			},
		},
		PortMappings: awsxEcs.TaskDefinitionPortMappingArray{
			awsxEcs.TaskDefinitionPortMappingArgs{
				ContainerPort: pulumi.Int(port),
				HostPort:      pulumi.Int(port),
				Protocol:      pulumi.StringPtr("tcp"),
			},
		},
		HealthCheck: &awsxEcs.TaskDefinitionHealthCheckArgs{
			Command: pulumi.ToStringArray([]string{"CMD-SHELL", "curl --fail http://localhost/fakeintake/health"}),
			// Explicitly set the following 3 parameters to their default values.
			// Because otherwise, `pulumi up` wants to recreate the task definition even when it is not needed.
			Interval: pulumi.IntPtr(30),
			Retries:  pulumi.IntPtr(3),
			Timeout:  pulumi.IntPtr(5),
		},
		DockerLabels: pulumi.StringMap{
			"com.datadoghq.ad.checks": pulumi.String(utils.JSONMustMarshal(
				map[string]interface{}{
					"openmetrics": map[string]interface{}{
						"init_config": map[string]interface{}{},
						"instances": []map[string]interface{}{
							{
								"openmetrics_endpoint": "http://%%host%%/metrics",
								"namespace":            "fakeintake",
								"metrics": []string{
									".*",
								},
							},
						},
					},
					"http_check": map[string]interface{}{
						"init_config": map[string]interface{}{},
						"instances": []map[string]interface{}{
							{
								"name": "health",
								"url":  "http://%%host%%/fakeintake/health",
							},
							{
								"name": "metrics query",
								"url":  "http://%%host%%/fakeintake/payloads?endpoint=/api/v2/series",
							},
							{
								"name": "logs query",
								"url":  "http://%%host%%/fakeintake/payloads?endpoint=/api/v2/logs",
							},
						},
					},
				}),
			),
		},
		LogConfiguration: ecs.GetFirelensLogConfiguration(pulumi.String("fakeintake"), pulumi.String("fakeintake"), apiKeySSMParamName),
	}
}

func buildFakeIntakeURL(scheme, host, path string, port int) string {
	url := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   path,
	}
	return url.String()
}
