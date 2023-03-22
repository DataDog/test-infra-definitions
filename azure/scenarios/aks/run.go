package aks

import (
	"github.com/DataDog/test-infra-definitions/azure"
	"github.com/DataDog/test-infra-definitions/azure/aks"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	env, err := azure.NewEnvironment(ctx)
	if err != nil {
		return err
	}

	_, kubeConfig, err := aks.NewCluster(env, "aks", nil)
	if err != nil {
		return err
	}

	ctx.Export("kubeconfig", kubeConfig)
	return nil
}
