package aws

import (
	config "github.com/vboulineau/pulumi-definitions/common/config"
)

type Environment interface {
	config.Environment
	APIKeySSMParamName() string

	ECSExecKMSKeyID() string
	ECSTaskExecutionRole() string
	ECSTaskRole() string

	AssignPublicIP() bool
	DefaultSubnet() string
	DefaultSecurityGroups() []string
}
