package azure

const (
	sandboxEnv = "az/sandbox"
)

type environmentDefault struct {
	azure   azureProvider
	ddInfra ddInfra
}

type azureProvider struct {
	tenantID       string
	subscriptionID string
}

type ddInfra struct {
	defaultResourceGroup   string
	defaultVNet            string
	defaultSubnet          string
	defaultSecurityGroup   string
	defaultInstanceType    string
	defaultARMInstanceType string
}

func getEnvironmentDefault(envName string) environmentDefault {
	switch envName {
	case sandboxEnv:
		return sandboxDefault()
	default:
		panic("Unknown environment: " + envName)
	}
}

func sandboxDefault() environmentDefault {
	return environmentDefault{
		azure: azureProvider{
			tenantID:       "4d3bac44-0230-4732-9e70-cc00736f0a97",
			subscriptionID: "8c56d827-5f07-45ce-8f2b-6c5001db5c6f",
		},
		ddInfra: ddInfra{
			defaultResourceGroup:   "datadog-agent-testing",
			defaultVNet:            "/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/datadog-agent-testing/providers/Microsoft.Network/virtualNetworks/default-vnet",
			defaultSubnet:          "/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/datadog-agent-testing/providers/Microsoft.Network/virtualNetworks/default-vnet/subnets/default-subnet",
			defaultSecurityGroup:   "/subscriptions/8c56d827-5f07-45ce-8f2b-6c5001db5c6f/resourceGroups/datadog-agent-testing/providers/Microsoft.Network/networkSecurityGroups/default",
			defaultInstanceType:    "Standard_B4ms",
			defaultARMInstanceType: "Standard_D4ps_v5",
		},
	}
}
