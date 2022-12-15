package config

import (
	"errors"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	sdkconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	ddMicroVMNamespace = "ddMicroVM"
)

type DDMicroVMConfig struct {
	Ctx           *pulumi.Context
	MicroVMConfig *sdkconfig.Config
}

func NewMicroVMConfig(ctx *pulumi.Context) DDMicroVMConfig {
	return DDMicroVMConfig{
		Ctx:           ctx,
		MicroVMConfig: sdkconfig.New(ctx, ddMicroVMNamespace),
	}
}

func (e *DDMicroVMConfig) GetStringWithDefault(config *sdkconfig.Config, paramName string, defaultValue string) string {
	val, err := config.Try(paramName)
	if err == nil {
		return val
	}

	if !errors.Is(err, sdkconfig.ErrMissingVar) {
		e.Ctx.Log.Error(fmt.Sprintf("Parameter %s not parsable, err: %v, will use default value: %v", paramName, err, defaultValue), nil)
	}

	return defaultValue
}
