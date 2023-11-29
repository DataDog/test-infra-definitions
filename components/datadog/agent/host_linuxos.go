package agent

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
)

type agentLinuxManager struct {
	targetOS os.OS
}

func newLinuxManager(targetOS os.OS) agentOSManager {
	return &agentLinuxManager{targetOS: targetOS}
}

func (am *agentLinuxManager) getInstallCommand(version agentparams.PackageVersion) (string, error) {
	if version.PipelineID != "" {
		testEnvVars := []string{}
		testEnvVars = append(testEnvVars, "TESTING_APT_URL=apttesting.datad0g.com")
		// apt testing repo
		// TESTING_APT_REPO_VERSION="pipeline-xxxxx-a7 7"
		testEnvVars = append(testEnvVars, fmt.Sprintf(`TESTING_APT_REPO_VERSION="%v-a7-%s 7"`, version.PipelineID, am.targetOS.Descriptor().Architecture))
		testEnvVars = append(testEnvVars, "TESTING_YUM_URL=yumtesting.datad0g.com")
		// yum testing repo
		// TESTING_YUM_VERSION_PATH="testing/pipeline-xxxxx-a7/7"
		testEnvVars = append(testEnvVars, fmt.Sprintf("TESTING_YUM_VERSION_PATH=testing/%v-a7/7", version.PipelineID))
		commandLine := strings.Join(testEnvVars, " ")

		return fmt.Sprintf(
			`DD_API_KEY=%%s %v bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
			commandLine,
			"install_script_agent7.sh"), nil
	}

	commandLine := fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%v ", version.Major)

	if version.Minor != "" {
		commandLine += fmt.Sprintf("DD_AGENT_MINOR_VERSION=%v ", version.Minor)
	}

	if version.BetaChannel {
		commandLine += "REPO_URL=datad0g.com DD_AGENT_DIST_CHANNEL=beta "
	}

	return fmt.Sprintf(
		`DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
		commandLine,
		"install_script.sh"), nil
}

func (am *agentLinuxManager) getAgentConfigFolder() string {
	return "/etc/datadog-agent"
}
