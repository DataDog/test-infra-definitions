package agent

import (
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentwithoperatorparams"
	"github.com/DataDog/test-infra-definitions/components/datadog/apps/dda"
	"github.com/DataDog/test-infra-definitions/components/datadog/operator"
	"github.com/DataDog/test-infra-definitions/components/datadog/operatorparams"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
)

func NewDDAWithOperator(e config.Env, resourceName string, kubeProvider *kubernetes.Provider, operatorOpts []operatorparams.Option, ddaOptions ...agentwithoperatorparams.Option) (*KubernetesAgent, error) {
	return components.NewComponent(e, resourceName, func(comp *KubernetesAgent) error {
		ddaParams, err := agentwithoperatorparams.NewParams(ddaOptions...)
		if err != nil {
			return err
		}

		operatorComp, err := operator.NewOperator(e, resourceName, kubeProvider, operatorOpts...)

		if err != nil {
			return err
		}

		_, ddaRef, err := dda.K8sAppDefinition(e, kubeProvider, ddaParams.Namespace, ddaParams.FakeIntake, ddaParams.KubeletTLSVerify, e.Ctx().Stack(), ddaParams.DDAConfig, utils.PulumiDependsOn(operatorComp))

		if err != nil {
			return err
		}

		baseName := "dda-linux"
		appVersion := ddaRef.AppVersion
		apiVersion := ddaRef.Version

		comp.LinuxNodeAgent, err = componentskube.NewKubernetesObjRef(e, baseName+"-nodeAgent", ddaParams.Namespace, "Pod", appVersion, apiVersion.ToStringOutput(), map[string]string{"app": baseName + "-datadog"})

		if err != nil {
			return err
		}

		comp.LinuxClusterAgent, err = componentskube.NewKubernetesObjRef(e, baseName+"-clusterAgent", ddaParams.Namespace, "Pod", appVersion, apiVersion, map[string]string{
			"app": baseName + "-datadog-cluster-agent",
		})

		if err != nil {
			return err
		}

		comp.LinuxClusterChecks, err = componentskube.NewKubernetesObjRef(e, baseName+"-clusterChecks", ddaParams.Namespace, "Pod", appVersion, apiVersion, map[string]string{
			"app": baseName + "-datadog-clusterchecks",
		})

		if err != nil {
			return err
		}

		return nil
	})
}
