package k8s

import (
	"encoding/json"
)

func jsonMustMarshal(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}
