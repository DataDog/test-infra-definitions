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

func (h *HostUpdater) Export(ctx *pulumi.Context, out *HostUpdaterOutput) error {
	return components.Export(ctx, h, out)
}

// NewHostUpdater creates a new instance of a on-host Updater
func NewHostUpdater(e *config.CommonEnvironment, host *remoteComp.Host, options ...agentparams.Option) (*HostUpdater, error) {
	hostInstallComp, err := components.NewComponent(*e, host.Name(), func(comp *HostUpdater) error {
		comp.namer = e.CommonNamer().WithPrefix(comp.Name())
		comp.host = host

		params, err := agentparams.NewParams(e, options...)
		if err != nil {
			return err
		}

		err = comp.installUpdater(params, pulumi.Parent(comp))
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

func (h *HostUpdater) installUpdater(params *agentparams.Params, baseOpts ...pulumi.ResourceOption) error {
	arch := h.host.OS.Descriptor().Architecture
	testAPTRepoVersion := fmt.Sprintf(`DD_TEST_APT_REPO_VERSION="%v-u7-%s 7"`, params.Version.PipelineID, arch)
	installCmdStr := fmt.Sprintf(`export %v && bash -c %s`, testAPTRepoVersion, installScript)
	_, err := h.host.OS.Runner().Command(
		h.namer.ResourceName("install-updater"),
		&command.Args{
			Create: pulumi.Sprintf(installCmdStr),
		}, baseOpts...)
	return err
}
