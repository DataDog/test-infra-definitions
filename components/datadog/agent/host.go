package agent

import (
	"fmt"
	"path"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	"github.com/DataDog/test-infra-definitions/components/os"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// HostAgent is an installer for the Agent on a remote host
type HostAgent struct {
	pulumi.ResourceState
	components.Component

	namer    namer.Namer
	manager  agentOSManager
	targetOS os.OS

	// Currently we don't have anything to export or expose to other components?
}

// NewHostAgent creates a new instance of a on-host Agent
func NewHostAgent(e *config.CommonEnvironment, vm *remoteComp.Host, options ...agentparams.Option) (*HostAgent, error) {
	hostInstallComp, err := components.NewComponent(*e, vm.Name(), func(comp *HostAgent) error {
		comp.namer = e.CommonNamer.WithPrefix(comp.Name())
		comp.manager = getOSManager(vm.OS)
		comp.targetOS = vm.OS

		params, err := agentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		err = comp.installAgent(e, params, pulumi.Parent(comp))
		if err != nil {
			return err
		}

		return nil
	}, pulumi.Parent(vm), pulumi.DeletedWith(vm))
	if err != nil {
		return nil, err
	}

	return hostInstallComp, nil
}

func (h *HostAgent) installAgent(env *config.CommonEnvironment, params *agentparams.Params, baseOpts ...pulumi.ResourceOption) error {
	installCmdStr, err := h.manager.getInstallCommand(params.Version)
	if err != nil {
		return err
	}

	installCmd, err := h.targetOS.Runner().Command(
		h.namer.ResourceName("install-agent"),
		&command.Args{
			Create: pulumi.Sprintf(installCmdStr, env.AgentAPIKey()),
		}, baseOpts...)
	if err != nil {
		return err
	}

	configFiles := make(map[string]pulumi.StringInput)
	for _, input := range []struct{ path, content string }{
		{"datadog.yaml", params.AgentConfig},
		{"system-probe.yaml", params.SystemProbeConfig},
		{"security-agent.yaml", params.SecurityAgentConfig},
	} {
		_, content, err := h.updateConfig(env, input.path, input.content, params.ExtraAgentConfig, utils.MergeOptions(baseOpts, utils.PulumiDependsOn(installCmd))...)
		if err != nil {
			return err
		}

		configFiles[input.path] = content
	}

	_, intgHash, err := h.installIntegrationsAndFiles(params.Integrations, params.Files, utils.MergeOptions(baseOpts, utils.PulumiDependsOn(installCmd))...)
	if err != nil {
		return err
	}

	// Restart the agent when the HostInstall itself is done, which is normally when all childs are done
	// So we cannot use baseOpts
	_, err = h.restartAgent(
		pulumi.Array{configFiles["datadog.yaml"], configFiles["system-probe.yaml"], configFiles["security-agent.yaml"], pulumi.String(intgHash)},
		utils.PulumiDependsOn(h),
	)
	return err
}

func (h *HostAgent) updateConfig(
	env *config.CommonEnvironment,
	configPath string,
	configContent string,
	extraAgentConfig []pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*remote.Command, pulumi.StringInput, error) {
	var err error

	configFullPath := path.Join(h.manager.getAgentConfigFolder(), configPath)
	pulumiAgentString := pulumi.String(configContent).ToStringOutput()
	// If core agent, set api key and extra configs
	if configPath == "datadog.yaml" {
		for _, extraConfig := range extraAgentConfig {
			pulumiAgentString = pulumi.Sprintf("%v\n%v", pulumiAgentString, extraConfig)
		}
		pulumiAgentString = pulumi.Sprintf("api_key: %v\n%v", env.AgentAPIKey(), pulumiAgentString)
	}

	copyCmd, err := h.targetOS.FileManager().CopyInlineFile(pulumiAgentString, configFullPath, true, opts...)
	if err != nil {
		return nil, pulumiAgentString, err
	}

	return copyCmd, pulumiAgentString, nil
}

func (h *HostAgent) restartAgent(triggers pulumi.ArrayInput, opts ...pulumi.ResourceOption) (*remote.Command, error) {
	return h.targetOS.ServiceManger().EnsureRunning("datadog-agent", triggers, opts...)
}

func (h *HostAgent) installIntegrationsAndFiles(
	integrations map[string]*agentparams.FileDefinition,
	files map[string]*agentparams.FileDefinition,
	opts ...pulumi.ResourceOption,
) ([]*remote.Command, string, error) {
	allCommands := make([]*remote.Command, 0)
	var parts []string

	// filePath is absolute path from params.WithFile but relative from params.WithIntegration
	for filePath, fileDef := range integrations {
		configFolder := h.manager.getAgentConfigFolder()
		fullPath := path.Join(configFolder, filePath)

		cmd, err := h.writeFileDefinition(fullPath, fileDef.Content, fileDef.UseSudo, opts...)
		if err != nil {
			return nil, "", err
		}
		allCommands = append(allCommands, cmd)
		parts = append(parts, filePath, fileDef.Content)
	}

	for fullPath, fileDef := range files {
		if !h.targetOS.FileManager().IsPathAbsolute(fullPath) {
			return nil, "", fmt.Errorf("failed to write file: \"%s\" is not an absolute filepath", fullPath)
		}

		cmd, err := h.writeFileDefinition(fullPath, fileDef.Content, fileDef.UseSudo, opts...)
		if err != nil {
			return nil, "", err
		}
		allCommands = append(allCommands, cmd)
		parts = append(parts, fullPath, fileDef.Content)
	}

	return allCommands, utils.StrHash(parts...), nil
}

func (h *HostAgent) writeFileDefinition(
	fullPath string,
	content string,
	useSudo bool,
	opts ...pulumi.ResourceOption,
) (*remote.Command, error) {
	// create directory, if it does not exist
	dirCommand, err := h.targetOS.FileManager().CreateDirectoryForFile(fullPath, useSudo, opts...)
	if err != nil {
		return nil, err
	}

	copyCmd, err := h.targetOS.FileManager().CopyInlineFile(pulumi.String(content), fullPath, useSudo, utils.MergeOptions(opts, utils.PulumiDependsOn(dirCommand))...)
	if err != nil {
		return nil, err
	}
	return copyCmd, nil
}
