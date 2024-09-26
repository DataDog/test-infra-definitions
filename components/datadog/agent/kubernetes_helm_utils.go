package agent

import "encoding/json"

// TODO: Remove these defaults when kubernetes_resource_labels_as_tags and kubernetes_resource_annotations_as_tags are added to the helm chart

type KubernetesResourcesMetadataAsTags map[string]map[string]string

func (k KubernetesResourcesMetadataAsTags) toJSONString() string {
	bytes, err := json.Marshal(k)
	if err != nil {
		return ""
	}

	return string(bytes)
}

func getResourcesLabelsAsTags() KubernetesResourcesMetadataAsTags {
	return KubernetesResourcesMetadataAsTags{
		"deployments.apps": {"x-team": "team"},
		"pods":             {"x-parent-type": "domain"},
		"namespaces":       {"kubernetes.io/metadata.name": "metadata-name"},
		"nodes":            {"kubernetes.io/os": "os", "kubernetes.io/arch": "arch"},
	}
}

func getResourcesAnnotationsAsTags() KubernetesResourcesMetadataAsTags {
	return KubernetesResourcesMetadataAsTags{
		"deployments.apps": {"x-sub-team": "sub-team"},
		"pods":             {"x-parent-name": "parent-name"},
		"namespaces":       {"related_email": "mail"},
	}
}
