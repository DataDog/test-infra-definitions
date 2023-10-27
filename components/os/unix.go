package os

import (
	"fmt"
	"strings"
)

type Unix struct{}

func NewUnix() *Unix {
	return &Unix{}
}

func (*Unix) GetAgentConfigFolder() string { return "/etc/datadog-agent" }

func (*Unix) CheckIsAbsPath(path string) bool {
	return strings.HasPrefix(path, "/")
}

func (*Unix) GetAgentInstallCmd(version AgentVersion) (string, error) {
	if version.PipelineID != "" {
		return getUnixInstallFormatString("install_script_agent7.sh", version), nil
	}
	return getUnixInstallFormatString("install_script.sh", version), nil
}

func (*Unix) GetType() Type {
	return UnixType
}

func (*Unix) GetRunAgentCmd(parameters string) string {
	return "sudo datadog-agent " + parameters
}

func getUnixInstallFormatString(scriptName string, version AgentVersion) string {
	if version.PipelineID != "" {
		testEnvVars := []string{}
		testEnvVars = append(testEnvVars, "TESTING_APT_URL=apttesting.datad0g.com")
		// apt testing repo
		// TESTING_APT_REPO_VERSION="pipeline-xxxxx-a7 7"
		testEnvVars = append(testEnvVars, fmt.Sprintf(`TESTING_APT_REPO_VERSION="%v-a7 7"`, version.PipelineID))
		testEnvVars = append(testEnvVars, "TESTING_YUM_URL=yumtesting.datad0g.com")
		// yum testing repo
		// TESTING_YUM_VERSION_PATH="testing/pipeline-xxxxx-a7/7"
		testEnvVars = append(testEnvVars, fmt.Sprintf("TESTING_YUM_VERSION_PATH=testing/%v-a7/7", version.PipelineID))
		commandLine := strings.Join(testEnvVars, " ")

		return fmt.Sprintf(
			`DD_API_KEY=%%s %v bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
			commandLine,
			scriptName)
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
		scriptName)
}
