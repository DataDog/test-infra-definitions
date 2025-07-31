package openshiftvm

import (
	"log"

	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components/datadog/agent/helm"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/cpustress"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/mutatedbyadmissioncontroller"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/nginx"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/prometheus"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/redis"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/tracegen"
	fakeintakeComp "github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"github.com/DataDog/test-infra-definitions/components/datadog/kubernetesagentparams"
	"github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/DataDog/test-infra-definitions/components/kubernetes/vpa"
	"github.com/DataDog/test-infra-definitions/components/os"
	resGcp "github.com/DataDog/test-infra-definitions/resources/gcp"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/compute"
	"github.com/DataDog/test-infra-definitions/scenarios/gcp/fakeintake"
	kubernetesNewProvider "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
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
	// Building Kubernetes provider for OpenShift
	openshiftKubeProvider, err := kubernetesNewProvider.NewProvider(ctx, gcpEnv.Namer.ResourceName("openshift-k8s-provider"), &kubernetesNewProvider.ProviderArgs{
		Kubeconfig:            openshiftCluster.KubeConfig,
		EnableServerSideApply: pulumi.BoolPtr(true),
		DeleteUnreachable:     pulumi.BoolPtr(true),
	})
	if err != nil {
		return err
	}

	vpaCrd, err := vpa.DeployCRD(&gcpEnv, openshiftKubeProvider)
	if err != nil {
		return err
	}
	dependsOnVPA := utils.PulumiDependsOn(vpaCrd)

	var fakeIntake *fakeintakeComp.Fakeintake
	if gcpEnv.AgentUseFakeintake() {
		fakeIntakeOptions := []fakeintake.Option{
			fakeintake.WithMemory(2048),
		}
		if gcpEnv.InfraShouldDeployFakeintakeWithLB() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithLoadBalancer())
		}
		if gcpEnv.AgentUseDualShipping() {
			fakeIntakeOptions = append(fakeIntakeOptions, fakeintake.WithoutDDDevForwarding())
		}

		if fakeIntake, err = fakeintake.NewVMInstance(gcpEnv, fakeIntakeOptions...); err != nil {
			return err
		}

		if err := fakeIntake.Export(gcpEnv.Ctx(), nil); err != nil {
			return err
		}
	}

	var dependsOnDDAgent pulumi.ResourceOption
	log.Println("[DEBUG] dependsOnDDAgent initialized as nil")

	// Deploy the agent
	if gcpEnv.AgentDeploy() {
		log.Println("[DEBUG] Agent deployment enabled, starting agent deployment")
		customValues := `
datadog:
  kubelet:
    tlsVerify: false
  apm:
    portEnabled: true
    socketEnabled: false
agents:
  enabled: true
  tolerations:
    # Deploy Agents on master nodes
    - effect: NoSchedule
      key: node-role.kubernetes.io/master
      operator: Exists
    # Deploy Agents on infra nodes
    - effect: NoSchedule
      key: node-role.kubernetes.io/infra
      operator: Exists
    # Tolerate disk pressure
    - effect: NoSchedule
      key: node.kubernetes.io/disk-pressure
      operator: Exists
  useHostNetwork: true
  replicas: 1
  podSecurity:
    securityContextConstraints:
      create: true
clusterAgent:
  resources:
    limits:
      cpu: 300m
      memory: 400Mi
    requests:
      cpu: 150m
      memory: 300Mi
  enabled: true
  podSecurity:
    securityContextConstraints:
      create: true`

		k8sAgentOptions := make([]kubernetesagentparams.Option, 0)
		k8sAgentOptions = append(
			k8sAgentOptions,
			kubernetesagentparams.WithNamespace("datadog-openshift"),
			kubernetesagentparams.WithHelmValues(customValues),
			kubernetesagentparams.WithClusterName(openshiftCluster.ClusterName),
		)
		if fakeIntake != nil {
			k8sAgentOptions = append(
				k8sAgentOptions,
				kubernetesagentparams.WithFakeintake(fakeIntake),
			)
		}

		if gcpEnv.AgentUseDualShipping() {
			k8sAgentOptions = append(k8sAgentOptions, kubernetesagentparams.WithDualShipping())
		}

		k8sAgentComponent, err := helm.NewKubernetesAgent(&gcpEnv, gcpEnv.Namer.ResourceName("datadog-agent"), openshiftKubeProvider, k8sAgentOptions...)

		if err != nil {
			log.Printf("[ERROR] Failed to create Kubernetes agent: %v", err)
			return err
		}
		log.Println("[DEBUG] Kubernetes agent component created successfully")

		if err := k8sAgentComponent.Export(gcpEnv.Ctx(), nil); err != nil {
			log.Printf("[ERROR] Failed to export Kubernetes agent component: %v", err)
			return err
		}
		log.Println("[DEBUG] Kubernetes agent component exported successfully")

		dependsOnDDAgent = utils.PulumiDependsOn(k8sAgentComponent)
		log.Println("[DEBUG] dependsOnDDAgent set to depend on k8sAgentComponent")
	} else {
		log.Println("[DEBUG] Agent deployment disabled, dependsOnDDAgent remains nil")
	}

	// Deploy testing workload
	if gcpEnv.TestingWorkloadDeploy() {
		if _, err := redis.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-redis", true /* for DDM */, dependsOnVPA, dependsOnDDAgent); err != nil {
			return err
		}
		if _, err := prometheus.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-prometheus"); err != nil {
			return err
		}
		if _, err := mutatedbyadmissioncontroller.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-mutated", "workload-mutated-lib-injection" /* for admission */, dependsOnDDAgent); err != nil {
			return err
		}
		if _, err := cpustress.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-cpustress"); err != nil {
			return err
		}
		if _, err := nginx.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-nginx", "", true /* for DDM */, dependsOnVPA, dependsOnDDAgent); err != nil {
			return err
		}
		if _, err := tracegen.K8sAppDefinition(&gcpEnv, openshiftKubeProvider, "workload-tracegen"); err != nil {
			return err
		}

	}

	return nil
}
