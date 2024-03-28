package utils

import "gopkg.in/yaml.v3"

func MergeYAMLString(oldValues string, newValues string) (string, error) {
	if oldValues == "" {
		return newValues, nil
	}

	if newValues == "" {
		return oldValues, nil
	}

	var oldValuesYAML map[string]interface{}
	var newValuesYAML map[string]interface{}

	err := yaml.Unmarshal([]byte(oldValues), &oldValuesYAML)
	if err != nil {
		return "", err
	}

	err = yaml.Unmarshal([]byte(newValues), &newValuesYAML)

	if err != nil {
		return "", err
	}

	mergedValuesYAML := mergeMaps(oldValuesYAML, newValuesYAML)

	mergedValues, err := yaml.Marshal(mergedValuesYAML)

	return string(mergedValues), err
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
