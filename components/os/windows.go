package os

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/components/command"
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

func (*Windows) GetAgentConfigFolder() string { return `C:\ProgramData\Datadog` }

func (*Windows) CreatePackageManager(*command.Runner) (command.PackageManager, error) {
	return nil, errors.New("package manager is not supported on Windows")
}

func (*Windows) GetAgentInstallCmd(version AgentVersion) (string, error) {
	url, err := getAgentURL(version)
	if err != nil {
		return "", err
	}

	localFilename := `C:\datadog-agent.msi`

	// Disable the progress as it slow downs the download.
	cmd := "$ProgressPreference = 'SilentlyContinue'"
	cmd += fmt.Sprintf("; Invoke-WebRequest %v -OutFile %v", url, localFilename)
	// Use `if ($?) { .. }` to get an error if the download fail.
	cmd += fmt.Sprintf(`; if ($?) { Start-Process -Wait msiexec -ArgumentList '/qn /i %v APIKEY="%%v" SITE="datadoghq.com"'}`, localFilename)
	return cmd, nil
}

func (*Windows) GetType() Type {
	return WindowsType
}

func (*Windows) GetRunAgentCmd(parameters string) string {
	return `& "$env:ProgramFiles\Datadog\Datadog Agent\bin\agent.exe" ` + parameters
}

func getWindowsRepositoryURL(version AgentVersion) string {
	baseURL := "https://ddagent-windows-stable.s3.amazonaws.com"
	if version.Repository == "staging" {
		baseURL = "https://s3.amazonaws.com/dd-agent-mstesting"
	}

	if version.Channel != "stable" {
		return fmt.Sprintf("%v/builds/%v", baseURL, version.Channel)
	}
	return baseURL
}

func getAgentURL(version AgentVersion) (string, error) {
	minor := strings.ReplaceAll(version.Minor, "~", "-")
	fullVersion := fmt.Sprintf("%v.%v", version.Major, minor)

	finder, err := newAgentURLFinder(fmt.Sprintf("%v/installers_v2.json", getWindowsRepositoryURL(version)))
	if err != nil {
		return "", err
	}

	if version.Repository == "staging" {
		url, err := finder.findVersion(fullVersion)
		if err != nil {
			// Try to handle custom build
			minor = strings.TrimSuffix(minor, "-1")
			return fmt.Sprintf("%v/ddagent-cli-%v.%v.msi", getWindowsRepositoryURL(version), version.Major, minor), nil
		}

		return url, nil
	}

	fullVersion += "-1"
	if version.Minor == "" { // Use latest
		if fullVersion, err = finder.getLatestVersion(); err != nil {
			return "", err
		}
	}

	return finder.findVersion(fullVersion)
}

type agentURLFinder struct {
	versions     map[string]interface{}
	installerURL string
}

func newAgentURLFinder(installerURL string) (*agentURLFinder, error) {
	resp, err := http.Get(installerURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	values := make(map[string]interface{})
	if err = json.Unmarshal(body, &values); err != nil {
		return nil, err
	}

	versions, err := getKey[map[string]interface{}](values, "datadog-agent")
	if err != nil {
		return nil, err
	}
	return &agentURLFinder{versions: versions, installerURL: installerURL}, nil
}

func (f *agentURLFinder) getLatestVersion() (string, error) {
	var versions []string
	for version := range f.versions {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return "", errors.New("no version found")
	}
	return versions[len(versions)-1], nil
}

func (f *agentURLFinder) findVersion(fullVersion string) (string, error) {
	version, err := getKey[map[string]interface{}](f.versions, fullVersion)
	if err != nil {
		return "", fmt.Errorf("the Agent version %v cannot be found at %v: %v", fullVersion, f.installerURL, err)
	}

	arch, err := getKey[map[string]interface{}](version, "x86_64")
	if err != nil {
		return "", fmt.Errorf("cannot find `x86_64` for Agent version %v at %v: %v", fullVersion, f.installerURL, err)
	}

	url, err := getKey[string](arch, "url")
	if err != nil {
		return "", fmt.Errorf("cannot find `url` for Agent version %v at %v: %v", fullVersion, f.installerURL, err)
	}

	return url, nil
}

func getKey[T any](m map[string]interface{}, keyName string) (T, error) {
	var t T
	abstractValue, ok := m[keyName]
	if !ok {
		return t, fmt.Errorf("cannot find the key %v", keyName)
	}

	value, ok := abstractValue.(T)
	if !ok {
		return t, fmt.Errorf("%v doesn't have the right type: %v", keyName, reflect.TypeOf(t))
	}
	return value, nil
}
