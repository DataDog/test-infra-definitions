package ec2

import (
	"fmt"
	"os"
	"strings"

	componentsos "github.com/DataDog/test-infra-definitions/components/os"
)

func getWindowsOpenSSHUserData(publicKeyPath string) (string, error) {
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}
	return buildAWSPowerShellUserData(
			componentsos.SetupSSHScriptContent,
			windowsPowerShellArgument{name: "authorizedKey", value: string(publicKey)},
		),
		nil
}

type windowsPowerShellArgument struct {
	name  string
	value string
}

func (a windowsPowerShellArgument) String() string {
	return fmt.Sprintf("-%s %s", a.name, a.value)
}

func buildAWSPowerShellUserData(scriptContent string, arguments ...windowsPowerShellArgument) string {
	scriptLines := strings.Split(scriptContent, "\n")
	userDataLines := make([]string, 0, len(scriptLines)+6+len(arguments))
	userDataLines = append(userDataLines, "<powershell>")
	for _, line := range scriptLines {
		// indent script lines by one tab
		userDataLines = append(userDataLines, fmt.Sprintf("		%s", line))
	}
	userDataLines = append(userDataLines, "</powershell>")
	userDataLines = append(userDataLines, "<persist>true</persist>")
	if len(arguments) > 0 {
		// You can specify one or more PowerShell arguments with the <powershellArguments> tag.
		// If no arguments are passed, EC2Launch and EC2Launch v2 add the following argument by default:
		// -ExecutionPolicy Unrestricted
		argumentsWithDefaults := make([]windowsPowerShellArgument, len(arguments)+1)
		argumentsWithDefaults[0] = windowsPowerShellArgument{name: "ExecutionPolicy", value: "Unrestricted"}
		copy(argumentsWithDefaults[1:], arguments)
		argumentsLine := fmt.Sprintf("<powershellArguments>%s</powershellArguments>", windowsArgumentsToString(argumentsWithDefaults))
		userDataLines = append(userDataLines, argumentsLine)
	}
	return strings.Join(userDataLines, "\n")
}

func windowsArgumentsToString(arguments []windowsPowerShellArgument) string {
	argumentStrings := make([]string, len(arguments))
	for i, arg := range arguments {
		argumentStrings[i] = arg.String()
	}
	return strings.Join(argumentStrings, " ")
}
