package agent

import (
	"fmt"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DefaultMajorVersion = "7"

type agentLinuxManager struct {
	targetOS os.OS
}

func newLinuxManager(host *remoteComp.Host) agentOSManager {
	return &agentLinuxManager{targetOS: host.OS}
}

func (am *agentLinuxManager) getInstallCommand(version agentparams.PackageVersion, _ []string) (string, error) {
	var commandLine string
	testEnvVars := []string{}

	if version.PipelineID != "" {
		testEnvVars = append(testEnvVars, "TESTING_APT_URL=apttesting.datad0g.com")
		// apt testing repo
		// TESTING_APT_REPO_VERSION="pipeline-xxxxx-a7 7"
		testEnvVars = append(testEnvVars, fmt.Sprintf(`TESTING_APT_REPO_VERSION="pipeline-%[1]v-a%[2]v-%[3]s %[2]v"`, version.PipelineID, version.Major, am.targetOS.Descriptor().Architecture))
		testEnvVars = append(testEnvVars, "TESTING_YUM_URL=yumtesting.datad0g.com")
		// yum testing repo
		// TESTING_YUM_VERSION_PATH="testing/pipeline-xxxxx-a7/7"
		testEnvVars = append(testEnvVars, fmt.Sprintf("TESTING_YUM_VERSION_PATH=testing/pipeline-%[1]v-a%[2]v/%[2]v", version.PipelineID, version.Major))
	} else {
		testEnvVars = append(testEnvVars, fmt.Sprintf("DD_AGENT_MAJOR_VERSION=%v", version.Major))

		if version.Minor != "" {
			testEnvVars = append(testEnvVars, fmt.Sprintf("DD_AGENT_MINOR_VERSION=%v", version.Minor))
		}

		if version.Channel != "" && version.Channel != agentparams.StableChannel {
			testEnvVars = append(testEnvVars, "REPO_URL=datad0g.com")
			testEnvVars = append(testEnvVars, fmt.Sprintf("DD_AGENT_DIST_CHANNEL=%s", version.Channel))
		}
	}

	commandLine = strings.Join(testEnvVars, " ")

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
