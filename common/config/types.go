package config

// Currently we cannot set structured objects through Automation API
// https://github.com/pulumi/pulumi/issues/5506
type EnvironmentConfig struct {
	Config Config `yaml:"config"`
}

type Config struct {
	AWS     AWS     `yaml:"aws"`
	DDInfra DDInfra `yaml:"ddinfra"`
	DDAgent DDAgent `yaml:"ddagent"`
}

type AWS struct {
	Region string `yaml:"region"`
}

type DDInfra struct {
	AWS AWSDDInfra `yaml:"aws"`
	ECS ECS        `yaml:"ecs"`
}

type DDAgent struct {
	Deploy bool `yaml:"deploy"`
}

type AWSDDInfra struct {
	DefaultVPCID          string   `yaml:"defaultVPCID"`
	DefaultSubnets        []string `yaml:"defaultSubnets"`
	DefaultSecurityGroups []string `yaml:"defaultSecurityGroups"`
	DefaultInstanceType   string   `yaml:"defaultInstanceType"`
	DefaultKeyPairName    string   `yaml:"defaultKeyPairName,omitempty"`
}

type ECS struct {
	ExecKMSKeyID               string `yaml:"execKMSKeyID"`
	TaskExecutionRole          string `yaml:"taskExecutionRole"`
	TaskRole                   string `yaml:"taskRole"`
	ServiceAllocatePublicIP    bool   `yaml:"serviceAllocatePublicIP"`
	FargateCapacityProvider    bool   `yaml:"fargateCapacityProvider"`
	LinuxECSOptimizedNodeGroup bool   `yaml:"linuxECSOptimizedNodeGroup"`
	LinuxBottlerocketNodeGroup bool   `yaml:"linuxBottlerocketNodeGroup"`
	WindowsLTSCNodeGroup       bool   `yaml:"windowsLTSCNodeGroup"`
}
