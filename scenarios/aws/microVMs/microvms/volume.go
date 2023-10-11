package microvms

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LibvirtVolume interface {
	SetupLibvirtVMVolume(ctx *pulumi.Context, runner *Runner, providerFn LibvirtProviderFn, isLocal bool, depends []pulumi.Resource) (pulumi.Resource, error)
	UnderlyingImage() *filesystemImage
	FullResourceName(string) string
	Key() string
	Pool() LibvirtPool
}

type filesystemImage struct {
	imageName   string
	imagePath   string
	imageSource string
}

type volume struct {
	filesystemImage
	pool        LibvirtPool
	volumeKey   string
	volumeXML   pulumi.StringOutput
	volumeNamer namer.Namer
}

func generateVolumeKey(poolPath string, volName string) string {
	return fmt.Sprintf("%s/%s", poolPath, volName)
}

func NewLibvirtVolume(pool LibvirtPool, fsImage filesystemImage, xmlDataFn func(string) pulumi.StringOutput, volNamerFn func(string) namer.Namer) LibvirtVolume {
	volKey := generateVolumeKey(pool.Path(), fsImage.imageName)
	return &volume{
		filesystemImage: fsImage,
		volumeKey:       volKey,
		volumeXML:       xmlDataFn(volKey),
		volumeNamer:     volNamerFn(volKey),
		pool:            pool,
	}
}

func remoteLibvirtVolume(v *volume, runner *Runner, depends []pulumi.Resource) (pulumi.Resource, error) {
	var baseVolumeReady pulumi.Resource

	volumeXMLPath := fmt.Sprintf("/tmp/volume-%s.xml", v.filesystemImage.imageName)
	volXMLWrittenArgs := command.Args{
		Create: pulumi.Sprintf("echo \"%s\" > %s", v.volumeXML, volumeXMLPath),
		Delete: pulumi.Sprintf("rm -f %s", volumeXMLPath),
	}
	// XML write does not need to depend on anything other than the instance being ready.
	// Instance state is handled by the runner automatically.
	volXMLWritten, err := runner.Command(
		v.volumeNamer.ResourceName("write-vol-xml"),
		&volXMLWrittenArgs,
	)
	if err != nil {
		return baseVolumeReady, err
	}

	depends = append(depends, volXMLWritten)

	baseVolumeReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh vol-create %s %s", v.pool.Name(), volumeXMLPath),
		Delete: pulumi.Sprintf("virsh vol-delete %s --pool %s", v.volumeKey, v.pool.Name()),
		Sudo:   true,
	}

	baseVolumeReady, err = runner.Command(v.volumeNamer.ResourceName("build-libvirt-basevolume"), &baseVolumeReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return baseVolumeReady, err
	}

	return baseVolumeReady, err
}

func localLibvirtVolume(v *volume, ctx *pulumi.Context, providerFn LibvirtProviderFn, depends []pulumi.Resource) (pulumi.Resource, error) {
	var stgvolReady pulumi.Resource

	provider, err := providerFn()
	if err != nil {
		return stgvolReady, err
	}

	stgvolReady, err = libvirt.NewVolume(ctx, v.volumeNamer.ResourceName("build-libvirt-basevolume"), &libvirt.VolumeArgs{
		Name:   pulumi.String(v.filesystemImage.imageName),
		Pool:   pulumi.String(v.pool.Name()),
		Source: pulumi.String(v.filesystemImage.imagePath),
		Xml: libvirt.VolumeXmlArgs{
			Xslt: v.volumeXML,
		},
	}, pulumi.Provider(provider), pulumi.DependsOn(depends))
	if err != nil {
		return stgvolReady, err
	}

	return stgvolReady, nil
}

func (v *volume) SetupLibvirtVMVolume(ctx *pulumi.Context, runner *Runner, providerFn LibvirtProviderFn, isLocal bool, depends []pulumi.Resource) (pulumi.Resource, error) {
	if isLocal {
		return localLibvirtVolume(v, ctx, providerFn, depends)
	}

	return remoteLibvirtVolume(v, runner, depends)
}

func (v *volume) UnderlyingImage() *filesystemImage {
	return &v.filesystemImage
}

func (v *volume) FullResourceName(name string) string {
	return v.volumeNamer.ResourceName(name)
}

func (v *volume) Key() string {
	return v.volumeKey
}

func (v *volume) Pool() LibvirtPool {
	return v.pool
}
