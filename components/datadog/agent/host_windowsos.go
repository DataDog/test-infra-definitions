package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type agentWindowsManager struct {
	host *remoteComp.Host
}

func newWindowsManager(host *remoteComp.Host) agentOSManager {
	return &agentWindowsManager{host: host}
}

func (am *agentWindowsManager) getInstallCommand(version agentparams.PackageVersion, additionalInstallParameters []string) (string, error) {
	url, err := getAgentURL(version)
	if err != nil {
		return "", err
	}

	logFilePath := "C:\\install.log"
	logParamIdx := slices.IndexFunc(additionalInstallParameters, func(s string) bool {
		return strings.HasPrefix(s, "/log")
	})
	if logParamIdx < 0 {
		additionalInstallParameters = append(additionalInstallParameters, fmt.Sprintf("/log %s", logFilePath))
	} else {
		// "/log C:\mycustomlog.txt" -> "C:\mycustomlog.txt"
		paramParts := strings.Split(additionalInstallParameters[logParamIdx], " ")
		if len(paramParts) != 2 {
			return "", fmt.Errorf("/log parameter was malformed, must be '/log <path_to_log_file>'")
		}
		logFilePath = paramParts[1]
	}

	localFilename := `C:\datadog-agent.msi`
	cmd := fmt.Sprintf(`
$ProgressPreference = 'SilentlyContinue';
$ErrorActionPreference = 'Stop';
for ($i=0; $i -lt 3; $i++) {
	try {
		(New-Object Net.WebClient).DownloadFile('%s','%s')
	} catch {
		if ($i -eq 2) {
			throw
		}
	}
};
$exitCode = (Start-Process -Wait msiexec -PassThru -ArgumentList '/qn /i %s APIKEY=%%s %s').ExitCode
Get-Content %s
Exit $exitCode 
`, url, localFilename, localFilename, strings.Join(additionalInstallParameters, " "), logFilePath)
	return cmd, nil
}

func (am *agentWindowsManager) getAgentConfigFolder() string {
	return `C:\ProgramData\Datadog`
}

func (am *agentWindowsManager) restartAgentServices(transform command.Transformer, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	// TODO: When we introduce Namer in components, we should use it here.
	cmdName := am.host.Name() + "-" + "restart-agent"
	cmdArgs := command.Args{
		Create: pulumi.String(`Start-Process "$($env:ProgramFiles)\Datadog\Datadog Agent\bin\agent.exe" -Wait -ArgumentList restart-service`),
	}

	// If a transform is provided, use it to modify the command name and args
	if transform != nil {
		cmdName, cmdArgs = transform(cmdName, cmdArgs)
	}

	return am.host.OS.Runner().Command(cmdName, &cmdArgs, opts...)
}

func getAgentURL(version agentparams.PackageVersion) (string, error) {
	minor := strings.ReplaceAll(version.Minor, "~", "-")
	fullVersion := fmt.Sprintf("%v.%v", version.Major, minor)

	if version.PipelineID != "" {
		return getAgentURLFromPipelineID(version.PipelineID)
	}

	if version.BetaChannel {
		finder, err := newAgentURLFinder("https://s3.amazonaws.com/dd-agent-mstesting/builds/beta/installers_v2.json")
		if err != nil {
			return "", err
		}

		url, err := finder.findVersion(fullVersion)
		if err != nil {
			// Try to handle custom build
			minor = strings.TrimSuffix(minor, "-1")
			return fmt.Sprintf("https://s3.amazonaws.com/dd-agent-mstesting/builds/beta/ddagent-cli-%v.%v.msi", version.Major, minor), nil
		}

		return url, nil
	}

	finder, err := newAgentURLFinder("https://ddagent-windows-stable.s3.amazonaws.com/installers_v2.json")
	if err != nil {
		return "", err
	}

	if version.Minor == "" { // Use latest
		if fullVersion, err = finder.getLatestVersion(); err != nil {
			return "", err
		}
	} else {
		fullVersion += "-1"
	}

	return finder.findVersion(fullVersion)
}

func getAgentURLFromPipelineID(pipeline string) (string, error) {
	// FIXME: remove pipeline- from the pipelineID we do not want it for Windows
	pipelineID := strings.TrimPrefix(pipeline, "pipeline-")

	// TODO: Replace context.Background() with a Pulumi context.Context.
	// dd-agent-mstesting is a public bucket so we can use anonymous credentials
	config, err := awsConfig.LoadDefaultConfig(context.Background(), awsConfig.WithCredentialsProvider(aws.AnonymousCredentials{}))
	if err != nil {
		return "", err
	}

	s3Client := s3.NewFromConfig(config)

	result, err := s3Client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String("dd-agent-mstesting"),
		Prefix: aws.String(fmt.Sprintf("pipelines/A7/%v", pipelineID)),
	})
	if err != nil {
		return "", err
	}

	if len(result.Contents) <= 0 {
		return "", fmt.Errorf("no agent MSI found for pipeline %v", pipeline)
	}

	return "https://s3.amazonaws.com/dd-agent-mstesting/" + *result.Contents[0].Key, nil
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
