package os

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
)

type Windows struct {
	env config.Environment
}

func NewWindows(env config.Environment) *Windows {
	return &Windows{
		env: env,
	}
}

func (w *Windows) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(w.env, arch)
}

func (*Windows) GetServiceManager() *ServiceManager {
	return &ServiceManager{restartCmd: []string{`Start-Process "$($env:ProgramFiles)\Datadog\Datadog Agent\bin\agent.exe" -Wait -ArgumentList restart-service`}}
}

func (*Windows) GetAgentConfigPath() string { return `C:\ProgramData\Datadog\datadog.yaml` }

func (*Windows) GetAgentInstallCmd(version AgentVersion) string {
	var url string
	stable := "https://s3.amazonaws.com/ddagent-windows-stable"
	if version.Minor == "" { // Use latest
		url = fmt.Sprintf("%v/datadog-agent-%v-latest.amd64.msi", stable, version.Major)
	} else {
		if version.BetaChannel {
			// transform 41.0~rc.7-1 into 41.0-rc.7
			minor := strings.ReplaceAll(version.Minor, "~", "-")
			minor = strings.TrimSuffix(minor, "-1")
			url = fmt.Sprintf("https://s3.amazonaws.com/dd-agent-mstesting/builds/beta/ddagent-cli-%v.%v.msi", version.Major, minor)
		} else {
			url = fmt.Sprintf("%v/ddagent-cli-%v.%v.msi", stable, version.Major, version.Minor)
		}
	}

	localFilename := `C:\datadog-agent.msi`
	cmd := fmt.Sprintf("Invoke-WebRequest %v -OutFile %v", url, localFilename)
	// Use `if ($?) { .. }` to get an error if the download fail.
	cmd += fmt.Sprintf(`; if ($?) { Start-Process -Wait msiexec -ArgumentList '/qn /i %v APIKEY="%%v" SITE="datadoghq.com"'}`, localFilename)
	return cmd
}

func (*Windows) GetType() Type {
	return WindowsType
}
