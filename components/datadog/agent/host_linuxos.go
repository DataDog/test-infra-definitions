package agent

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type agentLinuxManager struct {
	targetOS os.OS
}

func newLinuxManager(host *remoteComp.Host) agentOSManager {
	return &agentLinuxManager{targetOS: host.OS}
}

func (am *agentLinuxManager) getInstallCommand(version agentparams.PackageVersion, _ []string) (string, error) {
	if version.PipelineID != "" {
		// apt testing repo
		// TESTING_APT_REPO_VERSION="pipeline-xxxxx-a7-arch 7"
		// yum testing repo
		// TESTING_YUM_VERSION_PATH="testing/pipeline-xxxxx-a7/7"
		return testingInstallCommand(fmt.Sprintf("pipeline-%v-a7-%s", version.PipelineID, am.targetOS.Descriptor().Architecture), fmt.Sprintf("pipeline-%v-a7", version.PipelineID)), nil
	}

	if version.CustomVersion != "" {
		return testingInstallCommand(version.CustomVersion, version.CustomVersion), nil
	}

	commandLine := fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%v ", version.Major)

	if version.Minor != "" {
		commandLine += fmt.Sprintf("DD_AGENT_MINOR_VERSION=%v ", version.Minor)
	}

	if version.Channel != "" && version.Channel != agentparams.StableChannel {
		commandLine += fmt.Sprintf("REPO_URL=datad0g.com DD_AGENT_DIST_CHANNEL=%s ", version.Channel)
	}

	return fmt.Sprintf(
		`for i in 1 2 3 4 5; do curl -fsSL https://s3.amazonaws.com/dd-agent/scripts/%v -o install-script.sh && break || sleep $((2**$i)); done &&  for i in 1 2 3; do DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash install-script.sh  && exit 0 || sleep $((2**$i)); done; exit 1`,
		fmt.Sprintf("install_script_agent%s.sh", version.Major),
		commandLine), nil
}

func (am *agentLinuxManager) getAgentConfigFolder() string {
	return "/etc/datadog-agent"
}

func (am *agentLinuxManager) restartAgentServices(transform command.Transformer, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return am.targetOS.ServiceManger().EnsureRestarted("datadog-agent", transform, opts...)
}

func testingInstallCommand(aptRepoVersion string, yumVersionPath string) string {
	testEnvVars := []string{}
	testEnvVars = append(testEnvVars, "TESTING_APT_URL=apttesting.datad0g.com")
	testEnvVars = append(testEnvVars, fmt.Sprintf(`TESTING_APT_REPO_VERSION="%v 7"`, aptRepoVersion))
	testEnvVars = append(testEnvVars, "TESTING_YUM_URL=yumtesting.datad0g.com")
	testEnvVars = append(testEnvVars, fmt.Sprintf("TESTING_YUM_VERSION_PATH=testing/%v/7", yumVersionPath))
	commandLine := strings.Join(testEnvVars, " ")

	return fmt.Sprintf(`for i in 1 2 3 4 5; do curl -fsSL https://s3.amazonaws.com/dd-agent/scripts/%v -o install-script.sh && break || sleep $((2**$i)); done &&  for i in 1 2 3; do DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash install-script.sh  && exit 0  || sleep $((2**$i)); done; exit 1`, "install_script_agent7.sh", commandLine)
}
