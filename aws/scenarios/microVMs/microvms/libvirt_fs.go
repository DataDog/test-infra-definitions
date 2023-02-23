package microvms

import (
	"fmt"
	"path/filepath"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const basefsName = "custom-fsbase"

type filesystemImage struct {
	imageName     string
	imagePath     string
	imageSource   string
	volumeKey     string
	volumeXML     pulumi.StringOutput
	volumeXMLPath string
	volumeNamer   namer.Namer
}

type LibvirtFilesystem struct {
	poolName      string
	poolXML       pulumi.StringOutput
	poolXMLPath   string
	images        []*filesystemImage
	baseVolumeMap map[string]*filesystemImage
	poolNamer     namer.Namer
}

func generatePoolPath(name string) string {
	return "/pool/" + name + "/"
}

func generateVolumeKey(pool, volName string) string {
	return generatePoolPath(pool) + volName
}

func rootFSDir() string {
	return filepath.Join(GetWorkingDirectory(), "rootfs")
}

func getImagePath(name string) string {
	return filepath.Join(rootFSDir(), name)
}

func NewLibvirtFSDistroRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) *LibvirtFilesystem {
	var images []*filesystemImage

	rc := resources.NewResourceCollection(vmset.Recipe)
	poolName := vmset.Name

	poolPath := generatePoolPath(poolName)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName: pulumi.String(poolName),
			resources.PoolPath: pulumi.String(poolPath),
		},
	)
	baseVolumeMap := make(map[string]*filesystemImage)

	for _, k := range vmset.Kernels {
		imageName := poolName + "-" + k.Tag
		volKey := generateVolumeKey(poolName, imageName)
		img := &filesystemImage{
			imageName:   imageName,
			imagePath:   k.Dir,
			imageSource: k.ImageSource,
			volumeKey:   volKey,
			volumeXML: rc.GetVolumeXML(
				map[string]pulumi.StringInput{
					resources.ImageName:  pulumi.String(imageName),
					resources.VolumeKey:  pulumi.String(volKey),
					resources.VolumePath: pulumi.String(volKey),
				},
			),
			volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
			volumeNamer:   namer.NewNamer(ctx, volKey),
		}
		images = append(images, img)
		baseVolumeMap[k.Tag] = img
	}

	return &LibvirtFilesystem{
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        images,
		baseVolumeMap: baseVolumeMap,
		poolNamer:     namer.NewNamer(ctx, poolName),
	}
}

func NewLibvirtFSCustomRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) *LibvirtFilesystem {
	baseVolumeMap := make(map[string]*filesystemImage)
	poolName := vmset.Name
	imageName := vmset.Img.ImageName

	rc := resources.NewResourceCollection(vmset.Recipe)
	poolPath := generatePoolPath(poolName)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName: pulumi.String(poolName),
			resources.PoolPath: pulumi.String(poolPath),
		},
	)
	volKey := generateVolumeKey(poolName, basefsName)

	img := &filesystemImage{
		imageName:   imageName,
		imagePath:   getImagePath(imageName),
		imageSource: vmset.Img.ImageSourceURI,
		volumeKey:   volKey,
		volumeXML: rc.GetVolumeXML(
			map[string]pulumi.StringInput{
				resources.ImageName:  pulumi.String(basefsName),
				resources.VolumeKey:  pulumi.String(volKey),
				resources.VolumePath: pulumi.String(volKey),
			},
		),
		volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
		volumeNamer:   namer.NewNamer(ctx, volKey),
	}
	for _, k := range vmset.Kernels {
		baseVolumeMap[k.Tag] = img
	}

	return &LibvirtFilesystem{
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        []*filesystemImage{img},
		baseVolumeMap: baseVolumeMap,
		poolNamer:     namer.NewNamer(ctx, poolName),
	}
}

func downloadRootfs(fs *LibvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, fsImage := range fs.images {
		downloadRootfsArgs := command.Args{
			Create: pulumi.Sprintf("wget -q %s -O %s", fsImage.imageSource, fsImage.imagePath),
			Delete: pulumi.Sprintf("rm -f %s", fsImage.imagePath),
		}

		res, err := runner.Command(fsImage.volumeNamer.ResourceName("download-rootfs"), &downloadRootfsArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}
		waitFor = append(waitFor, res)
	}

	return waitFor, nil
}

func extractRootfs(fs *LibvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, fsImage := range fs.images {

		extractTopLevelArchive := command.Args{
			Create: pulumi.Sprintf("tar -C %s -xzf %s", rootFSDir(), fsImage.imagePath),
			Delete: pulumi.Sprintf("rm -rf %s", fsImage.imagePath),
		}
		res, err := runner.Command(fsImage.volumeNamer.ResourceName("extract-base-volume-package"), &extractTopLevelArchive, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, res)
	}

	for _, fsImage := range fs.images {
		res, err := runner.Command(fsImage.volumeNamer.ResourceName("copy-ssh-key-file"), &command.Args{
			Create: pulumi.Sprintf("cp %s/ddvm_rsa /tmp && chown 1000:1000 /tmp/ddvm_rsa", rootFSDir()),
			Sudo:   true,
		}, pulumi.DependsOn(waitFor))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, res)
	}

	return waitFor, nil
}

func setupLibvirtVMSetPool(fs *LibvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolBuildReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-build %s", fs.poolName),
		Delete: pulumi.Sprintf("virsh pool-delete %s", fs.poolName),
		Sudo:   true,
	}
	poolStartReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-start %s", fs.poolName),
		Delete: pulumi.Sprintf("virsh pool-destroy %s", fs.poolName),
		Sudo:   true,
	}
	poolRefreshDoneArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-refresh %s", fs.poolName),
		Sudo:   true,
	}

	poolDefineReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-define %s", fs.poolXMLPath),
		Sudo:   true,
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

func setupLibvirtVMVolume(fs *LibvirtFilesystem, runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, fsImage := range fs.images {
		baseVolumeReadyArgs := command.Args{
			Create: pulumi.Sprintf("virsh vol-create %s %s", fs.poolName, fsImage.volumeXMLPath),
			Delete: pulumi.Sprintf("virsh vol-delete %s --pool %s", fsImage.volumeKey, fs.poolName),
			Sudo:   true,
		}
		uploadImageToVolumeReadyArgs := command.Args{
			Create: pulumi.Sprintf("virsh vol-upload %s %s --pool %s", fsImage.volumeKey, fsImage.imagePath, fs.poolName),
			Sudo:   true,
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

func (fs *LibvirtFilesystem) setupLibvirtFilesystem(runner *command.Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfsDone, err := downloadRootfs(fs, runner, depends)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	extractRootfsDone, err := extractRootfs(fs, runner, downloadRootfsDone)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolXMLWrittenArgs := command.Args{
		Create: pulumi.Sprintf("echo \"%s\" > %s", fs.poolXML, fs.poolXMLPath),
		Delete: pulumi.Sprintf("rm -f %s", fs.poolXMLPath),
	}
	poolXMLWritten, err := runner.Command(fs.poolNamer.ResourceName("write-pool-xml"), &poolXMLWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	var volXMLWrittenArgs command.Args
	var volumeXMLReady []pulumi.Resource
	for _, fsImage := range fs.images {
		volXMLWrittenArgs = command.Args{
			Create: pulumi.Sprintf("echo \"%s\" > %s", fsImage.volumeXML, fsImage.volumeXMLPath),
			Delete: pulumi.Sprintf("rm -f %s", fsImage.volumeXMLPath),
		}
		volXMLWritten, err := runner.Command(fsImage.volumeNamer.ResourceName("write-vol-xml"), &volXMLWrittenArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		volumeXMLReady = append(volumeXMLReady, volXMLWritten)
	}

	setupLibvirtVMPoolDone, err := setupLibvirtVMSetPool(fs, runner, []pulumi.Resource{poolXMLWritten})
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
