package config

const (
	configNamespace = "ddinfra"
)

func GetParamKey(paramName string) string {
	return configNamespace + ":" + paramName
}
