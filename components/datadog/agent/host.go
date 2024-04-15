package agent

import (
	"fmt"
	"path"

	"github.com/DataDog/datadog-agent/pkg/util/optional"
	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/common/utils"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	perms "github.com/DataDog/test-infra-definitions/components/datadog/agentparams/filepermissions"
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

		deps := append(params.ResourceOptions, pulumi.Parent(comp))
		err = comp.installAgent(e, params, deps...)
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
	installCmdStr, err := h.manager.getInstallCommand(params.Version, params.AdditionalInstallParameters)
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

	afterInstallOpts := utils.MergeOptions(baseOpts, utils.PulumiDependsOn(installCmd))
	configFiles := make(map[string]pulumi.StringInput)

	// Update core Agent
	_, content, err := h.updateCoreAgentConfig(env, "datadog.yaml", pulumi.String(params.AgentConfig), params.ExtraAgentConfig, params.SkipAPIKeyInConfig, afterInstallOpts...)
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
	// Behind the scene `DependOn(h)` is transformed into `DependOn(<children>)`, the ComponentResource is skipped in the process.
	// With resources, Pulumi works in the following order:
	// Create -> Replace -> Delete.
	// The `DependOn` order is evaluated separately for each of these phases.
	// Thus, when an integration is deleted, the `Create` of `restartAgentServices` is done as there's no other `Create` from other resources to wait for.
	// Then the `Delete` of `restartAgentServices` is done, which is not waiting for the `Delete` of the integration as the dependecy on `Delete` is in reverse order.
	//
	// For this reason we have another `restartAgentServices` in `installIntegrationConfigsAndFiles` that is triggered when an integration is deleted.
	_, err = h.manager.restartAgentServices(
		// Transformer used to add triggers to the restart command
		func(name string, args command.Args) (string, command.Args) {
			args.Triggers = pulumi.Array{configFiles["datadog.yaml"], configFiles["system-probe.yaml"], configFiles["security-agent.yaml"], pulumi.String(intgHash)}
			return name, args
		},
		utils.PulumiDependsOn(h),
	)
	return err
}

func (h *HostAgent) updateCoreAgentConfig(
	env *config.CommonEnvironment,
	configPath string,
	configContent pulumi.StringInput,
	extraAgentConfig []pulumi.StringInput,
	skipAPIKeyInConfig bool,
	opts ...pulumi.ResourceOption,
) (*remote.Command, pulumi.StringInput, error) {
	for _, extraConfig := range extraAgentConfig {
		configContent = pulumi.Sprintf("%v\n%v", configContent, extraConfig)
	}
	if !skipAPIKeyInConfig {
		configContent = pulumi.Sprintf("api_key: %v\n%v", env.AgentAPIKey(), configContent)
	}

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

	// Build hash beforehand as we need to pass it to the restart command
	for filePath, fileDef := range integrations {
		parts = append(parts, filePath, fileDef.Content)
	}
	for fullPath, fileDef := range files {
		parts = append(parts, fullPath, fileDef.Content)
	}
	hash := utils.StrHash(parts...)

	// Restart the agent when an integration is removed
	// See longer comment in `installAgent` for more details
	restartCmd, err := h.manager.restartAgentServices(
		// Use a transformer to inject triggers on intg hash and move `restart` command from `Create` to `Delete`
		// so that it's run after the `Delete` commands of the integrations.
		func(name string, args command.Args) (string, command.Args) {
			args.Triggers = pulumi.Array{pulumi.String(hash)}
			args.Delete = args.Create
			args.Create = nil
			return name + "-on-intg-removal", args
		})
	if err != nil {
		return nil, "", err
	}

	opts = utils.MergeOptions(opts, utils.PulumiDependsOn(restartCmd))

	// filePath is absolute path from params.WithFile but relative from params.WithIntegration
	for filePath, fileDef := range integrations {
		configFolder := h.manager.getAgentConfigFolder()
		fullPath := path.Join(configFolder, filePath)

		cmd, err := h.writeFileDefinition(fullPath, fileDef.Content, fileDef.UseSudo, fileDef.Permissions, opts...)
		if err != nil {
			return nil, "", err
		}
		allCommands = append(allCommands, cmd)
	}

	for fullPath, fileDef := range files {
		if !h.host.OS.FileManager().IsPathAbsolute(fullPath) {
			return nil, "", fmt.Errorf("failed to write file: \"%s\" is not an absolute filepath", fullPath)
		}

		cmd, err := h.writeFileDefinition(fullPath, fileDef.Content, fileDef.UseSudo, fileDef.Permissions, opts...)
		if err != nil {
			return nil, "", err
		}
		allCommands = append(allCommands, cmd)
	}

	return allCommands, hash, nil
}

func (h *HostAgent) writeFileDefinition(
	fullPath string,
	content string,
	useSudo bool,
	perms optional.Option[perms.FilePermissions],
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

	// Set permissions if any
	if value, found := perms.Get(); found {
		if cmd := value.SetupPermissionsCommand(fullPath); cmd != "" {
			return h.host.OS.Runner().Command(
				h.namer.ResourceName("set-permissions-"+fullPath, utils.StrHash(cmd)),
				&command.Args{
					Create: pulumi.String(cmd),
					Delete: pulumi.String(value.ResetPermissionsCommand(fullPath)),
					Update: pulumi.String(value.ResetPermissionsCommand(fullPath)),
				},
				utils.PulumiDependsOn(copyCmd))
		}
	}

	return copyCmd, nil
}
