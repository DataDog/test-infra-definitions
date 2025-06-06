package crcvm

import (
	"github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/os"
	"github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := gcp.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	// creating a base VM with Ubuntu 22.04
	osDesc := os.DescriptorFromString("ubuntu:22.04", os.UbuntuDefault)
	vm, err := compute.NewVM(env, "crc-vm", compute.WithOS(osDesc))
	if err != nil {
		return err
	}

	// crc cluster on the vm
	cluster, err := kubernetes.NewCrcCluster(&env, vm, "crc")
	if err != nil {
		return err
	}

	// Export?
	if err := vm.Export(ctx, nil); err != nil {
		return err
	}
	return cluster.Export(ctx, nil)
}
