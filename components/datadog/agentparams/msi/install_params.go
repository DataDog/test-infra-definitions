// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package msi

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/components/datadog/fakeintake"
	"reflect"
)

// InstallAgentParams are the parameters used for installing the Agent using msiexec.
type InstallAgentParams struct {
	AgentUser         string `installer_arg:"DDAGENTUSER_NAME"`
	AgentUserPassword string `installer_arg:"DDAGENTUSER_PASSWORD"`
	DdURL             string `installer_arg:"DD_URL"`
	InstallLogFile    string
}

// InstallAgentOption is an optional function parameter type for InstallAgentParams options
type InstallAgentOption = func(*InstallAgentParams)

// NewInstallParams instantiates a new InstallAgentParams and runs all the given InstallAgentOption
// Example usage:
//
//	awshost.WithAgentOptions(
//	  agentparams.WithAdditionalInstallParameters(
//		msiparams.NewInstallParams(
//			msiparams.WithAgentUser(fmt.Sprintf("%s\\%s", TestDomain, TestUser)),
//			msiparams.WithAgentUserPassword(TestPassword)))),
func NewInstallParams(msiInstallParams ...InstallAgentOption) []string {
	msiInstallAgentParams := &InstallAgentParams{}
	for _, o := range msiInstallParams {
		o(msiInstallAgentParams)
	}
	return msiInstallAgentParams.toArgs()
}

// ToArgs convert the params to a list of valid msi switches, based on the `installer_arg` tag
func (p *InstallAgentParams) toArgs() []string {
	var args []string
	typeOfMSIInstallAgentParams := reflect.TypeOf(*p)
	for fieldIndex := 0; fieldIndex < typeOfMSIInstallAgentParams.NumField(); fieldIndex++ {
		field := typeOfMSIInstallAgentParams.Field(fieldIndex)
		installerArg := field.Tag.Get("installer_arg")
		if installerArg != "" {
			installerArgValue := reflect.ValueOf(*p).FieldByName(field.Name).String()
			if installerArgValue != "" {
				args = append(args, fmt.Sprintf("%s=%s", installerArg, installerArgValue))
			}
		}
	}
	return args
}

// WithAgentUser specifies the DDAGENTUSER_NAME parameter.
func WithAgentUser(username string) InstallAgentOption {
	return func(i *InstallAgentParams) {
		i.AgentUser = username
	}
}

// WithAgentUserPassword specifies the DDAGENTUSER_PASSWORD parameter.
func WithAgentUserPassword(password string) InstallAgentOption {
	return func(i *InstallAgentParams) {
		i.AgentUserPassword = password
	}
}

// WithDdURL specifies the DD_URL parameter.
func WithDdURL(ddURL string) InstallAgentOption {
	return func(i *InstallAgentParams) {
		i.DdURL = ddURL
	}
}

// WithInstallLogFile specifies the file where to save the MSI install logs.
func WithInstallLogFile(logFileName string) InstallAgentOption {
	return func(i *InstallAgentParams) {
		i.InstallLogFile = logFileName
	}
}

// WithFakeIntake configures the Agent to use a fake intake URL.
func WithFakeIntake(fakeIntake *fakeintake.FakeintakeOutput) InstallAgentOption {
	return WithDdURL(fakeIntake.URL)
}
