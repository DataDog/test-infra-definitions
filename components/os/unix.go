package os

import (
	"fmt"

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
	if version.IsCustomImage {
		scriptName := "install_script_agent" + getAgentMajorVersion(version) + ".sh"
		return getUnixInstallFormatString(scriptName, version), nil
	}
	return getUnixInstallFormatString("install_script.sh", version), nil
}

func (*Unix) GetType() Type {
	return UnixType
}

func (*Unix) GetRunAgentCmd(parameters string) string {
	return "sudo datadog-agent " + parameters
}

func getAgentMajorVersion(version AgentVersion) string {
	return string(version.RepoBranch[len(version.RepoBranch)-1])
}

func getDefaultInstanceType(env config.Environment, arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return env.DefaultInstanceType()
	case ARM64Arch:
		return env.DefaultARMInstanceType()
	default:
		panic("Architecture not supportede")
	}
}

func getUnixInstallFormatString(scriptName string, version AgentVersion) string {
	if version.IsCustomImage {
		commandLine := fmt.Sprintf(`TESTING_APT_URL=apttesting.datad0g.com TESTING_APT_REPO_VERSION="%v %v" `, version.RepoBranch, getAgentMajorVersion(version))
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
