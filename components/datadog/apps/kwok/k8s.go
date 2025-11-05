package kwok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	componentskube "github.com/DataDog/test-infra-definitions/components/kubernetes"
)

const KwokRepo = "kubernetes-sigs/kwok"

func K8sAppDefinition(e config.Env, kubeProvider *kubernetes.Provider, opts ...pulumi.ResourceOption) (*componentskube.Workload, error) {
	opts = append(opts, pulumi.Provider(kubeProvider), pulumi.Parent(kubeProvider), pulumi.DeletedWith(kubeProvider))

	k8sComponent := &componentskube.Workload{}
	if err := e.Ctx().RegisterComponentResource("dd:apps", "kwok", k8sComponent, opts...); err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(k8sComponent))

	release, err := getLatestRelease(e.Ctx().Context())
	if err != nil {
		return nil, err
	}

	kwok, err := yaml.NewConfigFile(e.Ctx(), "kwok", &yaml.ConfigFileArgs{
		File: "https://github.com/" + KwokRepo + "/releases/download/" + release + "/kwok.yaml",
	}, opts...)
	if err != nil {
		return nil, err
	}

	if res := kwok.GetResource("apiextensions.k8s.io/v1/CustomResourceDefinition", "stages.kwok.x-k8s.io", ""); res != nil {
		opts = append(opts, utils.PulumiDependsOn(res))
	}

	if _, err := yaml.NewConfigFile(e.Ctx(), "kwok-stage", &yaml.ConfigFileArgs{
		File: "https://github.com/" + KwokRepo + "/releases/download/" + release + "/stage-fast.yaml",
	}, opts...); err != nil {
		return nil, err
	}

	return k8sComponent, nil
}

func getLatestRelease(ctx context.Context) (string, error) {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+KwokRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}

	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var r struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	return r.TagName, nil
}
