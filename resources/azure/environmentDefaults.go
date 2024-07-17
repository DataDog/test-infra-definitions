package azure

const (
	sandboxEnv = "az/agent-sandbox"
)

type environmentDefault struct {
	azure   azureProvider
	ddInfra ddInfra
}

type azureProvider struct {
	tenantID       string
	subscriptionID string
	location       string
}

type ddInfra struct {
	defaultResourceGroup   string
	defaultVNet            string
	defaultSubnet          string
	defaultSecurityGroup   string
	defaultInstanceType    string
	defaultARMInstanceType string
	aks                    ddInfraAks
}

type ddInfraAks struct {
	linuxKataNodeGroup bool
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
			tenantID:       "cc0b82f3-7c2e-400b-aec3-40a3d720505b",
			subscriptionID: "9972cab2-9e99-419b-a683-86bfa77b3df1",
			location:       "West US 2",
		},
		ddInfra: ddInfra{
			defaultResourceGroup:   "dd-agent-sandbox",
			defaultVNet:            "/subscriptions/9972cab2-9e99-419b-a683-86bfa77b3df1/resourceGroups/dd-agent-sandbox/providers/Microsoft.Network/virtualNetworks/dd-agent-sandbox",
			defaultSubnet:          "/subscriptions/9972cab2-9e99-419b-a683-86bfa77b3df1/resourceGroups/dd-agent-sandbox/providers/Microsoft.Network/virtualNetworks/dd-agent-sandbox/subnets/dd-agent-sandbox-private",
			defaultSecurityGroup:   "/subscriptions/9972cab2-9e99-419b-a683-86bfa77b3df1/resourceGroups/dd-agent-sandbox/providers/Microsoft.Network/networkSecurityGroups/appgategreen",
			defaultInstanceType:    "Standard_D4s_v5",  // Allows nested virtualization for kata runtimes
			defaultARMInstanceType: "Standard_D4ps_v5", // No azure arm instance supports nested virtualization
			aks: ddInfraAks{
				linuxKataNodeGroup: true,
			},
		},
	}
}
