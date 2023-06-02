package os

import (
	"fmt"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/config"
)

type Unix struct {
	env config.Environment
}

func NewUnix(env config.Environment) *Unix {
	return &Unix{
		env: env,
	}
}

func (u *Unix) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(u.env, arch)
}
func (*Unix) GetAgentConfigFolder() string { return "/etc/datadog-agent" }

func (*Unix) GetAgentInstallCmd(version AgentVersion) (string, error) {
	return getUnixInstallFormatString("install_script_agent7.sh", version), nil
}

func (*Unix) GetType() Type {
	return UnixType
}

func (*Unix) GetRunAgentCmd(parameters string) string {
	return "sudo datadog-agent " + parameters
}

func getDefaultInstanceType(env config.Environment, arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return env.DefaultInstanceType()
	case ARM64Arch:
		return env.DefaultARMInstanceType()
	default:
		panic("Architecture not supported")
	}
}

func getUnixRepositoryURL(version AgentVersion) string {
	if version.Repository == "staging" {
		return "datad0g.com"
	}
	return "datadoghq.com"
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

	if version.Repository != "" {
		commandLine += fmt.Sprintf("DD_REPO_URL=%v ", getUnixRepositoryURL(version))
	}

	if version.Channel != "" {
		commandLine += fmt.Sprintf("DD_AGENT_DIST_CHANNEL=%v ", version.Channel)
	}

	return fmt.Sprintf(
		`DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
		commandLine,
		scriptName)
}
