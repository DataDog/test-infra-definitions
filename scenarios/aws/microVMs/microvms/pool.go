package microvms

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LibvirtPool interface {
	SetupLibvirtPool(ctx *pulumi.Context, runner *Runner, providerFn LibvirtProviderFn, isLocal bool, depends []pulumi.Resource) ([]pulumi.Resource, error)
	Name() string
}

type globalLibvirtPool struct {
	poolName    string
	poolXML     pulumi.StringOutput
	poolXMLPath string
	poolNamer   namer.Namer
}

//func generateRamBackedPool() (map[string]pulumi.StringInput, string) {
//	poolName := libvirtResourceName(ctx.Stack(), "ram-pool")
//	return generatePool(poolName, generateRamPoolPath()), poolName
//}

func generateGlobalPoolPath(name string) string {
	return fmt.Sprintf("%s/libvirt/pools/%s", GetWorkingDirectory(), name)
}

func generateRamPoolPath() string {
	return fmt.Sprintf("%s/kmt-ramfs/", GetWorkingDirectory())
}

func NewGlobalLibvirtPool(ctx *pulumi.Context) LibvirtPool {
	poolName := libvirtResourceName(ctx.Stack(), "global-pool")
	rc := resources.NewResourceCollection(vmconfig.RecipeDefault)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName: pulumi.String(poolName),
			resources.PoolPath: pulumi.String(generateGlobalPoolPath(poolName)),
		},
	)

	return &globalLibvirtPool{
		poolName:    poolName,
		poolXML:     poolXML,
		poolXMLPath: fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		poolNamer:   libvirtResourceNamer(ctx, poolName),
	}
}

/*
Setup for remote pool and local pool is different for a number of reasons:
  - Libvirt pools and volumes on remote machines are setup using the virsh cli tool. This is because
    the pulumi-libvirt sdk always uploads the base volume image from the host (where pulumi runs) to the
    remote machine (where the micro-vms are setup).
    This is too inefficient for us. We would like for it to assume the images are already present on the remote
    machine. Therefore we create volumes using the virsh cli and we have to create the pools in the same way
    since we cannot pass the `pool` object, returned by the pulumi-libvirt api,  around in remote commands.
  - On the remote machine all commands are run with 'sudo' to simplify permission issues;
    we do not want to do this on the local machine. For local machines the pulumi-libvirt API works fine, since
    the target environment and the pulumi host are the same machine. It is simpler to use this API locally than
    have a complicated permissions setup.
*/
func remoteGlobalPool(p *globalLibvirtPool, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolBuildReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-build %s", p.poolName),
		Delete: pulumi.Sprintf("virsh pool-delete %s", p.poolName),
		Sudo:   true,
	}
	poolStartReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-start %s", p.poolName),
		Delete: pulumi.Sprintf("virsh pool-destroy %s", p.poolName),
		Sudo:   true,
	}
	poolRefreshDoneArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-refresh %s", p.poolName),
		Sudo:   true,
	}

	poolDefineReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-define %s", p.poolXMLPath),
		Sudo:   true,
	}

	poolDefineReady, err := runner.Command(p.poolNamer.ResourceName("define-libvirt-pool"), &poolDefineReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command(p.poolNamer.ResourceName("build-libvirt-pool"), &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReady, err := runner.Command(p.poolNamer.ResourceName("start-libvirt-pool"), &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command(p.poolNamer.ResourceName("refresh-libvirt-pool"), &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolRefreshDone}, nil
}

func localGlobalPool(ctx *pulumi.Context, p *globalLibvirtPool, providerFn LibvirtProviderFn, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	provider, err := providerFn()
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolReady, err := libvirt.NewPool(ctx, "create-libvirt-pool", &libvirt.PoolArgs{
		Type: pulumi.String("dir"),
		Name: pulumi.String(p.poolName),
		Path: pulumi.String(generateGlobalPoolPath(p.poolName)),
	}, pulumi.Provider(provider), pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolReady}, nil
}

func (p *globalLibvirtPool) SetupLibvirtPool(ctx *pulumi.Context, runner *Runner, providerFn LibvirtProviderFn, isLocal bool, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	if isLocal {
		return localGlobalPool(ctx, p, providerFn, depends)
	}

	return remoteGlobalPool(p, runner, depends)
}

func (p *globalLibvirtPool) Name() string {
	return p.poolName
}
