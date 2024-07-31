package utils

import "gopkg.in/yaml.v3"

func MergeYAML(oldValuesYamlContent string, newValuesYamlContent string) (string, error) {
	if oldValuesYamlContent == "" {
		return newValuesYamlContent, nil
	}

	if newValuesYamlContent == "" {
		return oldValuesYamlContent, nil
	}

	var oldValuesYAML map[string]interface{}
	var newValuesYAML map[string]interface{}

	err := yaml.Unmarshal([]byte(oldValuesYamlContent), &oldValuesYamlContent)
	if err != nil {
		return "", err
	}

	err = yaml.Unmarshal([]byte(newValuesYamlContent), &newValuesYAML)

	if err != nil {
		return "", err
	}

	mergedValuesYAML := MergeMaps(oldValuesYAML, newValuesYAML)

	mergedValues, err := yaml.Marshal(mergedValuesYAML)

	return string(mergedValues), err
}

func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
