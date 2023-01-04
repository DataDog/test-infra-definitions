package microVMs

import (
	"fmt"

	_ "embed"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microVMs/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const basefsName = "custom-fsbase"

type filesystemImage struct {
	imageName     string
	imagePath     string
	imageSource   string
	volumeKey     string
	volumeXML     string
	volumeXMLPath string
	volumeNamer   common.Namer
}

type libvirtFilesystem struct {
	poolName      string
	poolXML       string
	poolXMLPath   string
	images        []*filesystemImage
	baseVolumeMap map[string]*filesystemImage
	poolNamer     common.Namer
}

func generatePoolPath(name string) string {
	return "/pool/" + name + "/"
}

func generateVolumeKey(pool, volName string) string {
	return generatePoolPath(pool) + volName
}

func getImagePath(name string) string {
	return fmt.Sprintf("/tmp/%s", name)
}

func NewLibvirtFSDistroRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) *libvirtFilesystem {
	var images []*filesystemImage
	poolName := vmset.Name

	poolPath := generatePoolPath(poolName)
	poolXML := fmt.Sprintf(resources.GetRecipePoolTemplateOrDefault(vmset.Recipe), poolName, poolPath)
	baseVolumeMap := make(map[string]*filesystemImage)

	for _, k := range vmset.Kernels {
		imageName := poolName + "-" + k.Tag
		volKey := generateVolumeKey(poolName, imageName)
		img := &filesystemImage{
			imageName:     imageName,
			imagePath:     k.Dir,
			imageSource:   k.ImageSource,
			volumeKey:     volKey,
			volumeXML:     fmt.Sprintf(resources.GetRecipeVolumeTemplateOrDefault(vmset.Recipe), imageName, volKey, volKey),
			volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
			volumeNamer:   common.NewNamer(ctx, volKey),
		}
		images = append(images, img)
		baseVolumeMap[k.Tag] = img
	}

	return &libvirtFilesystem{
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        images,
		baseVolumeMap: baseVolumeMap,
		poolNamer:     common.NewNamer(ctx, poolName),
	}
}

func NewLibvirtFSCustomRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) *libvirtFilesystem {
	baseVolumeMap := make(map[string]*filesystemImage)
	poolName := vmset.Name
	imageName := vmset.Img.ImageName

	poolPath := generatePoolPath(poolName)
	poolXML := fmt.Sprintf(resources.GetRecipePoolTemplateOrDefault(vmset.Recipe), poolName, poolPath)
	volKey := generateVolumeKey(poolName, basefsName)

	img := &filesystemImage{
		imageName:     imageName,
		imagePath:     getImagePath(imageName),
		imageSource:   vmset.Img.ImageSourceURI,
		volumeKey:     volKey,
		volumeXML:     fmt.Sprintf(resources.GetRecipeVolumeTemplateOrDefault(vmset.Recipe), basefsName, volKey, volKey),
		volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
		volumeNamer:   common.NewNamer(ctx, volKey),
	}
	for _, k := range vmset.Kernels {
		baseVolumeMap[k.Tag] = img
	}

	return &libvirtFilesystem{
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        []*filesystemImage{img},
		baseVolumeMap: baseVolumeMap,
		poolNamer:     common.NewNamer(ctx, poolName),
	}
}

func downloadRootfs(fs *libvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, fsImage := range fs.images {
		downloadRootfsArgs := command.CommandArgs{
			Create: pulumi.String(
				fmt.Sprintf("wget -q %s -O %s", fsImage.imageSource, fsImage.imagePath),
			),
			Delete: pulumi.String(fmt.Sprintf("rm -f %s", fsImage.imagePath)),
		}

		res, err := runner.Command(fsImage.volumeNamer.ResourceName("download-rootfs"), &downloadRootfsArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}
		waitFor = append(waitFor, res)
	}

	return waitFor, nil
}

func extractRootfs(fs *libvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, fsImage := range fs.images {

		extractTopLevelArchive := command.CommandArgs{
			Create: pulumi.String(
				fmt.Sprintf("pushd /tmp; tar -xzf %s; popd;", fsImage.imagePath),
			),
			Delete: pulumi.String(
				fmt.Sprintf("rm -rf %s", fsImage.imagePath),
			),
		}
		res, err := runner.Command(fsImage.volumeNamer.ResourceName("extract-base-volume-package"), &extractTopLevelArchive, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, res)
	}

	return waitFor, nil
}

func setupLibvirtVMSetPool(fs *libvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
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

	poolDefineReadyArgs := command.CommandArgs{
		Create: pulumi.String(
			fmt.Sprintf("virsh pool-define %s", fs.poolXMLPath),
		),
		Sudo: true,
	}

	poolDefineReady, err := runner.Command(fs.poolNamer.ResourceName("define-libvirt-pool"), &poolDefineReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command(fs.poolNamer.ResourceName("build-libvirt-pool"), &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReady, err := runner.Command(fs.poolNamer.ResourceName("start-libvirt-pool"), &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command(fs.poolNamer.ResourceName("refresh-libvirt-pool"), &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolRefreshDone}, err
}

func setupLibvirtVMVolume(fs *libvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, fsImage := range fs.images {
		baseVolumeReadyArgs := command.CommandArgs{
			Create: pulumi.String(fmt.Sprintf("virsh vol-create %s %s", fs.poolName, fsImage.volumeXMLPath)),
			Delete: pulumi.String(fmt.Sprintf("virsh vol-delete %s --pool %s", fsImage.volumeKey, fs.poolName)),
			Sudo:   true,
		}
		uploadImageToVolumeReadyArgs := command.CommandArgs{
			Create: pulumi.String(
				fmt.Sprintf("virsh vol-upload %s %s --pool %s", fsImage.volumeKey, fsImage.imagePath, fs.poolName),
			),
			Sudo: true,
		}

		baseVolumeReady, err := runner.Command(fsImage.volumeNamer.ResourceName("build-libvirt-basevolume"), &baseVolumeReadyArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		uploadImageToVolumeReady, err := runner.Command(fsImage.volumeNamer.ResourceName("upload-libvirt-volume"), &uploadImageToVolumeReadyArgs, pulumi.DependsOn([]pulumi.Resource{baseVolumeReady}))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, uploadImageToVolumeReady)
	}

	return waitFor, nil
}

func (fs *libvirtFilesystem) setupLibvirtFilesystem(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
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
			fmt.Sprintf("echo \"%s\" > %s", fs.poolXML, fs.poolXMLPath),
		),
		Delete: pulumi.String(
			fmt.Sprintf("rm -f %s", fs.poolXMLPath),
		),
	}
	poolXmlWritten, err := runner.Command(fs.poolNamer.ResourceName("write-pool-xml"), &poolXmlWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	var volXmlWrittenArgs command.CommandArgs
	var volumeXMLReady []pulumi.Resource
	for _, fsImage := range fs.images {
		volXmlWrittenArgs = command.CommandArgs{
			Create: pulumi.String(
				fmt.Sprintf("echo \"%s\" > %s", fsImage.volumeXML, fsImage.volumeXMLPath),
			),
			Delete: pulumi.String(
				fmt.Sprintf("rm -f %s", fsImage.volumeXMLPath),
			),
		}
		volXmlWritten, err := runner.Command(fsImage.volumeNamer.ResourceName("write-vol-xml"), &volXmlWrittenArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		volumeXMLReady = append(volumeXMLReady, volXmlWritten)
	}

	setupLibvirtVMPoolDone, err := setupLibvirtVMSetPool(fs, runner, []pulumi.Resource{poolXmlWritten})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMVolumeDone, err := setupLibvirtVMVolume(fs, runner, append(
		append(volumeXMLReady, extractRootfsDone...), setupLibvirtVMPoolDone...),
	)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return setupLibvirtVMVolumeDone, nil
}
