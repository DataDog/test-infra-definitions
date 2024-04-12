package eks

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/resources/aws"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewENIConfigs(e aws.Environment, provider *kubernetes.Provider, subnets []aws.DDInfraEKSPodSubnets, securityGroups []string, opts ...pulumi.ResourceOption) (*yaml.ConfigGroup, error) {
	if len(subnets) == 0 {
		return nil, fmt.Errorf("subnets must not be empty")
	}

	objects := make([]map[string]interface{}, 0, len(subnets))
	for _, subnet := range subnets {
		objects = append(objects, map[string]interface{}{
			"apiVersion": "crd.k8s.amazonaws.com/v1alpha1",
			"kind":       "ENIConfig",
			"metadata": map[string]interface{}{
				"name": subnet.AZ,
			},
			"spec": map[string]interface{}{
				"securityGroups": securityGroups,
				"subnet":         subnet.SubnetID,
			},
		})
	}

	return yaml.NewConfigGroup(e.Ctx, e.Namer.ResourceName("eks-eni-configs"), &yaml.ConfigGroupArgs{
		Objs: objects,
	}, utils.MergeOptions(opts, pulumi.Providers(provider))...)
}
