package vm

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	resourcesAws "github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2vm"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := resourcesAws.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osType, err := getOSType(env.CommonEnvironment)
	if err != nil {
		return err
	}
	vm, err := ec2vm.NewEC2VMWithEnv(env, ec2params.WithOS(osType))
	if err != nil {
		return err
	}

	if vm.GetCommonEnvironment().AgentDeploy() {
		agentOptions := []func(*agentparams.Params) error{}
		if vm.GetCommonEnvironment().AgentUseFakeintake() {
			fakeintake, err := aws.NewEcsFakeintake(vm.Infra.GetAwsEnvironment())
			if err != nil {
				return err
			}
			agentOptions = append(agentOptions, agentparams.WithFakeintake(fakeintake))
		}

		_, err = agent.NewInstaller(vm, agentOptions...)
		return err
	}

	return nil
}

func getOSType(commonEnv *config.CommonEnvironment) (ec2os.Type, error) {
	var osType ec2os.Type
	osTypeStr := strings.ToLower(commonEnv.InfraOSFamily())
	switch osTypeStr {
	case "windows":
		osType = ec2os.WindowsOS
	case "ubuntu":
		osType = ec2os.UbuntuOS
	case "amazonlinux":
		osType = ec2os.AmazonLinuxOS
	case "debian":
		osType = ec2os.DebianOS
	case "redhat":
		osType = ec2os.RedHatOS
	case "suse":
		osType = ec2os.SuseOS
	case "fedora":
		osType = ec2os.FedoraOS
	case "":
		osType = ec2os.UbuntuOS // Default
	default:
		return osType, fmt.Errorf("the os type '%v' is not valid", osTypeStr)
	}
	return osType, nil
}
