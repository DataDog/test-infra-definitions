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

	paws "github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acmpca"
	clb "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/alb"
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
	sslPort       = 443
	cpu           = "256"
	memory        = "2048"
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

		ca, err := acmpca.NewCertificateAuthority(e.Ctx, namer.ResourceName("ca"), &acmpca.CertificateAuthorityArgs{
			Type: pulumi.StringPtr("ROOT"),
			CertificateAuthorityConfiguration: acmpca.CertificateAuthorityCertificateAuthorityConfigurationArgs{
				KeyAlgorithm:     pulumi.String("RSA_4096"),
				SigningAlgorithm: pulumi.String("SHA512WITHRSA"),
				Subject: acmpca.CertificateAuthorityCertificateAuthorityConfigurationSubjectArgs{
					CommonName: e.CommonNamer.DisplayName(64, pulumi.String("fakeintake CA")),
				},
			},
			RevocationConfiguration: acmpca.CertificateAuthorityRevocationConfigurationArgs{
				CrlConfiguration: acmpca.CertificateAuthorityRevocationConfigurationCrlConfigurationArgs{
					Enabled: pulumi.BoolPtr(false),
				},
				OcspConfiguration: acmpca.CertificateAuthorityRevocationConfigurationOcspConfigurationArgs{
					Enabled: pulumi.Bool(false),
				},
			},
			PermanentDeletionTimeInDays: pulumi.IntPtr(7),
		}, opts...)
		if err != nil {
			return nil, err
		}

		current, err := paws.GetPartition(e.Ctx, nil, nil)
		if err != nil {
			return nil, err
		}
		caCert, err := acmpca.NewCertificate(e.Ctx, namer.ResourceName("ca-cert"), &acmpca.CertificateArgs{
			CertificateAuthorityArn:   ca.Arn,
			CertificateSigningRequest: ca.CertificateSigningRequest,
			SigningAlgorithm:          pulumi.String("SHA512WITHRSA"),
			TemplateArn:               pulumi.String(fmt.Sprintf("arn:%v:acm-pca:::template/RootCACertificate/V1", current.Partition)),
			Validity: acmpca.CertificateValidityArgs{
				Value: pulumi.String("10"),
				Type:  pulumi.String("YEARS"),
			},
		}, opts...)
		if err != nil {
			return nil, err
		}

		if _, err = acmpca.NewCertificateAuthorityCertificate(e.Ctx, namer.ResourceName("ca-cert-cert"), &acmpca.CertificateAuthorityCertificateArgs{
			CertificateAuthorityArn: ca.Arn,
			Certificate:             caCert.Certificate,
			CertificateChain:        caCert.CertificateChain,
		}, opts...); err != nil {
			return nil, err
		}

		domainName := alb.LoadBalancer.DnsName().ApplyT(func(dnsName string) *string {
			if len(dnsName) <= 64 {
				return &dnsName
			}
			dummyDNSName := "fakeintake.datadoghq.com"
			return &dummyDNSName
		}).(pulumi.StringPtrOutput)
		cert, err := acm.NewCertificate(e.Ctx, namer.ResourceName("cert"), &acm.CertificateArgs{
			CertificateAuthorityArn: ca.Arn,
			KeyAlgorithm:            pulumi.String("RSA_2048"),
			DomainName:              domainName,
			SubjectAlternativeNames: pulumi.StringArray{
				alb.LoadBalancer.DnsName(),
			},
		}, opts...)
		if err != nil {
			return nil, err
		}

		if _, err = clb.NewListener(e.Ctx, namer.ResourceName("lb-https"), &clb.ListenerArgs{
			LoadBalancerArn: alb.LoadBalancer.Arn(),
			Port:            pulumi.IntPtr(sslPort),
			Protocol:        pulumi.StringPtr("HTTPS"),
			CertificateArn:  cert.Arn,
			DefaultActions: clb.ListenerDefaultActionArray{
				clb.ListenerDefaultActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: alb.DefaultTargetGroup.Arn(),
				},
			},
		}, opts...); err != nil {
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
				Value: pulumi.StringPtr("1536MiB"),
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
		fakeintakeURL := getFakeintakeHealthURL(ipAddress)
		err = backoff.Retry(func() error {
			e.Ctx.Log.Debug(fmt.Sprintf("getting fakeintake health at %s", fakeintakeURL), nil)
			resp, err := http.Get(fakeintakeURL)
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
