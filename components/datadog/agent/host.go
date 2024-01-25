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
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type HostAgentOutput struct {
	components.JSONImporter
}

// HostAgent is an installer for the Agent on a remote host
type HostAgent struct {
	pulumi.ResourceState
	components.Component

	namer   namer.Namer
	manager agentOSManager
	host    *remoteComp.Host

	// Currently we don't have anything to export or expose to other components?
}

func (h *HostAgent) Export(ctx *pulumi.Context, out *HostAgentOutput) error {
	return components.Export(ctx, h, out)
}

// NewHostAgent creates a new instance of a on-host Agent
func NewHostAgent(e *config.CommonEnvironment, host *remoteComp.Host, options ...agentparams.Option) (*HostAgent, error) {
	hostInstallComp, err := components.NewComponent(*e, host.Name(), func(comp *HostAgent) error {
		comp.namer = e.CommonNamer.WithPrefix(comp.Name())
		comp.host = host
		comp.manager = getOSManager(host)

		params, err := agentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		err = comp.installAgent(e, params, pulumi.Parent(comp))
		if err != nil {
			return err
		}

		return nil
	}, pulumi.Parent(host), pulumi.DeletedWith(host))
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

	installCmd, err := h.host.OS.Runner().Command(
		h.namer.ResourceName("install-agent"),
		&command.Args{
			Create: pulumi.Sprintf(installCmdStr, env.AgentAPIKey()),
		}, baseOpts...)
	if err != nil {
		return err
	}

	varddcmd, err := h.host.OS.Runner().Command(
		h.namer.ResourceName("var/run/datadog"),
		&command.Args{
			Create: pulumi.String("mkdir -p /var/run/datadog"),
			Sudo:   true,
		},
	)
	if err != nil {
		return err
	}

	_, err = h.host.OS.Runner().Command(
		h.namer.ResourceName("var/run/datadog perm"),
		&command.Args{
			Create: pulumi.String("chown dd-agent:dd-agent /var/run/datadog"),
			Sudo:   true,
		},
		utils.PulumiDependsOn(varddcmd),
		utils.PulumiDependsOn(installCmd),
	)
	if err != nil {
		return err
	}

	_, err = h.host.OS.Runner().Command(
		h.namer.ResourceName("dd-agent:docker group"),
		&command.Args{
			Create: pulumi.String("usermod -a -G docker dd-agent"),
			Sudo:   true,
		},
		utils.PulumiDependsOn(installCmd),
	)
	if err != nil {
		return err
	}

	afterInstallOpts := utils.MergeOptions(baseOpts, utils.PulumiDependsOn(installCmd))
	configFiles := make(map[string]pulumi.StringInput)

	// Update core Agent
	_, content, err := h.updateCoreAgentConfig(env, "datadog.yaml", pulumi.String(params.AgentConfig), params.ExtraAgentConfig, afterInstallOpts...)
	if err != nil {
		return err
	}
	configFiles["datadog.yaml"] = content

	// Update other Agents
	for _, input := range []struct{ path, content string }{
		{"system-probe.yaml", params.SystemProbeConfig},
		{"security-agent.yaml", params.SecurityAgentConfig},
	} {
		contentPulumiStr := pulumi.String(input.content)
		_, err := h.updateConfig(input.path, contentPulumiStr, afterInstallOpts...)
		if err != nil {
			return err
		}

		configFiles[input.path] = contentPulumiStr
	}

	_, intgHash, err := h.installIntegrationConfigsAndFiles(params.Integrations, params.Files, afterInstallOpts...)
	if err != nil {
		return err
	}

	// Restart the agent when the HostInstall itself is done, which is normally when all children are done
	_, err = h.manager.restartAgentServices(
		pulumi.Array{configFiles["datadog.yaml"], configFiles["system-probe.yaml"], configFiles["security-agent.yaml"], pulumi.String(intgHash)},
		utils.PulumiDependsOn(h),
	)
	return err
}

func (h *HostAgent) updateCoreAgentConfig(
	env *config.CommonEnvironment,
	configPath string,
	configContent pulumi.StringInput,
	extraAgentConfig []pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*remote.Command, pulumi.StringInput, error) {
	for _, extraConfig := range extraAgentConfig {
		configContent = pulumi.Sprintf("%v\n%v", configContent, extraConfig)
	}
	configContent = pulumi.Sprintf("api_key: %v\n%v", env.AgentAPIKey(), configContent)

	cmd, err := h.updateConfig(configPath, configContent, opts...)
	return cmd, configContent, err
}

func (h *HostAgent) updateConfig(
	configPath string,
	configContent pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*remote.Command, error) {
	var err error

	configFullPath := path.Join(h.manager.getAgentConfigFolder(), configPath)

	copyCmd, err := h.host.OS.FileManager().CopyInlineFile(configContent, configFullPath, true, opts...)
	if err != nil {
		return nil, err
	}

	return copyCmd, nil
}

func (h *HostAgent) installIntegrationConfigsAndFiles(
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
		if !h.host.OS.FileManager().IsPathAbsolute(fullPath) {
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
	dirCommand, err := h.host.OS.FileManager().CreateDirectoryForFile(fullPath, useSudo, opts...)
	if err != nil {
		return nil, err
	}

	copyCmd, err := h.host.OS.FileManager().CopyInlineFile(pulumi.String(content), fullPath, useSudo, utils.MergeOptions(opts, utils.PulumiDependsOn(dirCommand))...)
	if err != nil {
		return nil, err
	}
	return copyCmd, nil
}
