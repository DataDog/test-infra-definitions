package utils

import (
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

func MergeYAML(oldValuesYamlContent string, newValuesYamlContent string) (string, error) {
	if oldValuesYamlContent == "" {
		return newValuesYamlContent, nil
	}

	if newValuesYamlContent == "" {
		return oldValuesYamlContent, nil
	}

	var oldValuesYAML map[string]interface{}
	var newValuesYAML map[string]interface{}

	err := yaml.Unmarshal([]byte(oldValuesYamlContent), &oldValuesYAML)
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
	out := maps.Clone(a)
	for keyB, valueB := range b {
		// deep merge nested maps
		if valueB, ok := valueB.(map[string]interface{}); ok {
			if valueA, ok := out[keyB]; ok {
				if valueA, ok := valueA.(map[string]interface{}); ok {
					out[keyB] = MergeMaps(valueA, valueB)
					continue
				}
			}
		}
		// deep merge slices
		if valueB, ok := valueB.([]interface{}); ok {
			if valueA, ok := out[keyB]; ok {
				if valueA, ok := valueA.([]interface{}); ok {
					out[keyB] = append(valueA, valueB...)
					continue
				}
			}
		}
		out[keyB] = valueB
	}
	return out
}
