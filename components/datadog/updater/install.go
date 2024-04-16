package updater

import (
	_ "embed"
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/components/datadog/agentparams"
	remoteComp "github.com/DataDog/test-infra-definitions/components/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

//go:embed install_script.sh
var installScript string

type HostUpdaterOutput struct {
	components.JSONImporter
}

// HostUpdater is an installer for the Agent on a virtual machine
type HostUpdater struct {
	pulumi.ResourceState
	components.Component

	namer namer.Namer
	host  *remoteComp.Host
}

const installerPath = "/opt/datadog-installer/bin/installer/installer"
const latestOciUrlFormatString = "oci://docker.io/datadog/%s:latest"

func (h *HostUpdater) Export(ctx *pulumi.Context, out *HostUpdaterOutput) error {
	return components.Export(ctx, h, out)
}

// NewHostUpdater creates a new instance of a on-host Updater
func NewHostUpdater(e *config.CommonEnvironment, host *remoteComp.Host, options ...agentparams.Option) (*HostUpdater, error) {
	hostInstallComp, err := components.NewComponent(*e, host.Name(), func(comp *HostUpdater) error {
		comp.namer = e.CommonNamer.WithPrefix(comp.Name())
		comp.host = host

		params, err := agentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		err = comp.installUpdater(params, []string{}, pulumi.Parent(comp))
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

// NewHostUpdaterWithPacakges creates a new instance of a on-host Updater with packages bootstrap
func NewHostUpdaterWithPackages(e *config.CommonEnvironment, host *remoteComp.Host, packages []string, options ...agentparams.Option) (*HostUpdater, error) {
	hostInstallComp, err := components.NewComponent(*e, host.Name(), func(comp *HostUpdater) error {
		comp.namer = e.CommonNamer.WithPrefix(comp.Name())
		comp.host = host

		params, err := agentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		err = comp.installUpdater(params, packages, pulumi.Parent(comp))
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

func (h *HostUpdater) installUpdater(params *agentparams.Params, packages []string, baseOpts ...pulumi.ResourceOption) error {
	pipelineID := fmt.Sprintf("DD_PIPELINE_ID=%v", params.Version.PipelineID)
	agentConfig := pulumi.Sprintf("")
	for _, extraConfig := range params.ExtraAgentConfig {
		agentConfig = pulumi.Sprintf("%v\n%v", agentConfig, extraConfig)
	}
	agentConfig = pulumi.Sprintf("AGENT_CONFIG='%v'", agentConfig)
	installCmdStr := pulumi.Sprintf(`export %v %v && bash -c %s`, pipelineID, agentConfig, installScript)

	for _, pkg := range packages {
		installCmdStr = pulumi.Sprintf(
			"%v\nsudo %s bootstrap --url \"%s\"",
			installCmdStr,
			installerPath,
			fmt.Sprintf(latestOciUrlFormatString, pkg),
		)
	}

	_, err := h.host.OS.Runner().Command(
		h.namer.ResourceName("install-updater"),
		&command.Args{
			Create: installCmdStr,
		}, baseOpts...)
	return err
}
