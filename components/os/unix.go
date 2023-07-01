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
	// install_script_agent7.sh, despite its name, can also install Agent 6
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

func getUnixRepositoryParams(version AgentVersion) string {
	envVars := []string{}
	switch version.Repository {
	case TrialRepository, TestingRepository:
		aptChannel := version.Channel
		yumChannel := version.Channel
		if version.Repository == TestingRepository {
			aptChannel = fmt.Sprintf("pipeline-%v-a%v", version.PipelineID, version.Major)
			yumChannel = fmt.Sprintf("testing/pipeline-%v-a%v", version.PipelineID, version.Major)
		}

		envVars = append(envVars, fmt.Sprintf(`TESTING_APT_URL="apt%v.datad0g.com"`, version.Repository))
		envVars = append(envVars, fmt.Sprintf(`TESTING_APT_REPO_VERSION="%v %v"`, aptChannel, version.Major))
		envVars = append(envVars, fmt.Sprintf(`TESTING_YUM_URL="yum%v.datad0g.com"`, version.Repository))
		envVars = append(envVars, fmt.Sprintf(`TESTING_YUM_VERSION_PATH="%v/%v"`, yumChannel, version.Major))
	case StagingRepository:
		envVars = append(envVars, `DD_REPO_URL="datad0g.com"`)
		envVars = append(envVars, fmt.Sprintf(`DD_AGENT_DIST_CHANNEL="%v"`, version.Channel))
	case ProdRepository:
		envVars = append(envVars, `DD_REPO_URL="datadoghq.com"`)
		envVars = append(envVars, fmt.Sprintf(`DD_AGENT_DIST_CHANNEL="%v"`, version.Channel))
	}

	return strings.Join(envVars, " ")
}

func getUnixInstallFormatString(scriptName string, version AgentVersion) string {
	commandEnvVars := fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%v ", version.Major)

	if version.Minor != "" {
		commandEnvVars += fmt.Sprintf("DD_AGENT_MINOR_VERSION=%v ", version.Minor)
	}

	commandEnvVars += getUnixRepositoryParams(version)

	return fmt.Sprintf(
		`DD_API_KEY=%%s %v DD_INSTALL_ONLY=true bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/%v)"`,
		commandEnvVars,
		scriptName)
}
