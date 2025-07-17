package openshiftvm

import (
	"github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/os"
	resGcp "github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	gcpEnv, err := resGcp.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	osDesc := os.DescriptorFromString("redhat:9", os.RedHat9)
	vm, err := compute.NewVM(gcpEnv, "openshift",
		compute.WithOS(osDesc))
	if err != nil {
		return err
	}
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}

	openshiftCluster, err := kubernetes.NewOpenShiftCluster(&gcpEnv, vm, "openshift", gcpEnv.OpenShiftPullSecretPath())
	if err != nil {
		return err
	}
	if err := openshiftCluster.Export(ctx, nil); err != nil {
		return err
	}
	return openshiftCluster.Export(ctx, nil)
}
