package microVMs

import (
	"fmt"

	_ "embed"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const basefsName = "custom-fsbase"

//go:embed resources/volume.xml
var volumeXMLTemplate string

//go:embed resources/pool.xml
var poolXMLTemplate string

type libvirtFS struct {
	poolName    string
	poolXML     string
	volumeKey   string
	volumeXML   string
	imageName   string
	imageSource string
}

func generatePoolPath(name string) string {
	return "/pool/" + name + "/"
}

func generateVolumeKey(pool string) string {
	return generatePoolPath(pool) + basefsName
}

func getImagePath(name string) string {
	return fmt.Sprintf("/tmp/%s", name)
}

func NewLibvirtFS(poolName string, img *vmconfig.Image) *libvirtFS {
	poolPath := generatePoolPath(poolName)
	poolXML := fmt.Sprintf(poolXMLTemplate, poolName, poolPath)

	volKey := generateVolumeKey(poolName)
	volPath := volKey
	volumeXML := fmt.Sprintf(volumeXMLTemplate, basefsName, volKey, volPath)

	return &libvirtFS{
		poolName:    poolName,
		poolXML:     poolXML,
		volumeKey:   volKey,
		volumeXML:   volumeXML,
		imageName:   img.ImageName,
		imageSource: img.ImageSourceURI,
	}
}

func downloadRootfs(fs *libvirtFS, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	imagePath := getImagePath(fs.imageName)
	downloadRootfsArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("wget -q %s -O %s", fs.imageSource, imagePath),
		),
		Delete: pulumi.String(fmt.Sprintf("rm -f %s", imagePath)),
	}

	res, err := runner.Command("download-rootfs", &downloadRootfsArgs, pulumi.DependsOn(depends))
	return []pulumi.Resource{res}, err
}

func extractRootfs(fs *libvirtFS, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	extractTopLevelArchive := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("pushd /tmp; tar -xzf %s; popd;", getImagePath(fs.imageName)),
		),
		Delete: pulumi.String(
			fmt.Sprintf("rm -rf %s", getImagePath(fs.imageName)),
		),
	}
	res, err := runner.Command("extract-base-volume-package", &extractTopLevelArchive, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{res}, err
}

func setupLibvirtVMSetPool(fs *libvirtFS, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolBuildReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-build %s", fs.poolName),
		),
		Delete: pulumi.String(
			fmt.Sprintf("virsh pool-delete %s", fs.poolName),
		),
		Sudo: true,
	}
	poolStartReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-start %s", fs.poolName),
		),
		Delete: pulumi.String(
			fmt.Sprintf("virsh pool-destroy %s", fs.poolName),
		),
		Sudo: true,
	}
	poolRefreshDoneArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-refresh %s", fs.poolName),
		),
		Sudo: true,
	}

	poolDefineReady, err := runner.Command("define-libvirt-pool", &poolDefineReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command("build-libvirt-pool", &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReady, err := runner.Command("start-libvirt-pool", &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command("refresh-libvirt-pool", &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolRefreshDone}, err
}

func setupLibvirtVMVolume(fs *libvirtFS, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	baseVolumeReadyArgs := command.CommandArgs{
		Create: pulumi.String(fmt.Sprintf("virsh vol-create %s /tmp/volume.xml", fs.poolName)),
		Delete: pulumi.String(fmt.Sprintf("virsh vol-delete %s --pool %s", fs.volumeKey, fs.poolName)),
		Sudo:   true,
	}
	uploadImageToVolumeReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh vol-upload %s %s --pool %s", fs.volumeKey, getImagePath(fs.imageName), fs.poolName),
		),
		Sudo: true,
	}

	baseVolumeReady, err := runner.Command("build-libvirt-basevolume", &baseVolumeReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	uploadImageToVolumeReady, err := runner.Command("upload-libvirt-volume", &uploadImageToVolumeReadyArgs, pulumi.DependsOn([]pulumi.Resource{baseVolumeReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{uploadImageToVolumeReady}, err
}

func (fs *libvirtFS) setupLibvirtFilesystem(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfsDone, err := downloadRootfs(fs, runner, []pulumi.Resource{})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractRootfsDone, err := extractRootfs(fs, runner, downloadRootfsDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolXmlWrittenArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("echo \"%s\" > /tmp/pool.xml", fs.poolXML),
		),
		Delete: pulumi.String("rm -f /tmp/pool.xml"),
	}
	poolXmlWritten, err := runner.Command("write-pool-xml", &poolXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	volXmlWrittenArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("echo \"%s\" > /tmp/volume.xml", fs.volumeXML),
		),
		Delete: pulumi.String("rm -f /tmp/volume.xml"),
	}
	volXmlWritten, err := runner.Command("write-vol-xml", &volXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMPoolDone, err := setupLibvirtVMSetPool(fs, runner, []pulumi.Resource{poolXmlWritten})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMVolumeDone, err := setupLibvirtVMVolume(fs, runner, append(
		append([]pulumi.Resource{volXmlWritten}, extractRootfsDone...), setupLibvirtVMPoolDone...),
	)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return setupLibvirtVMVolumeDone, nil
}
