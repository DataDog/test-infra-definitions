package main

import (
	"log"

	k8s "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/resources/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		env, err := local.NewEnvironment(ctx)

		if err != nil {
			log.Fatalf("Failed to create environment: %v", err)
		}

		cluster, err := k8s.NewLocalCRCCluster(&env, "test-crc")
		if err != nil {
			return err
		}

		ctx.Export("clusterName", cluster.ClusterName)
		ctx.Export("kubeConfig", cluster.KubeConfig)

		return nil
	})
}
