package microvms

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/aws/scenarios/microVMs/vmconfig"
	"github.com/DataDog/test-infra-definitions/command"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const basefsName = "custom-fsbase"
const refreshFromEBS = "fio --filename=%s --rw=read --bs=64m --iodepth=32 --ioengine=libaio --direct=1 --name=volume-initialize"

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
	ctx           *pulumi.Context
	poolName      string
	poolXML       pulumi.StringOutput
	poolXMLPath   string
	images        []*filesystemImage
	baseVolumeMap map[string]*filesystemImage
	poolNamer     namer.Namer
}

func generatePoolPath(name string) string {
	return "/home/kernel-version-testing/libvirt/pools/" + name + "/"
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

func NewLibvirtFSDistroRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) (*LibvirtFilesystem, error) {
	var images []*filesystemImage

	rc := resources.NewResourceCollection(vmset.Recipe)
	poolName := vmset.Name

	poolPath := generatePoolPath(poolName)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName:     pulumi.String(poolName),
			resources.PoolPath:     pulumi.String(poolPath),
			resources.User:         pulumi.String(currentUser.Uid),
			resources.LibvirtGroup: pulumi.String(libvirtGroup.Gid),
		},
	)
	baseVolumeMap := make(map[string]*filesystemImage)

	for _, k := range vmset.Kernels {
		imageName := poolName + "-" + k.Tag
		volKey := generateVolumeKey(poolName, imageName)
		img := &filesystemImage{
			imageName:   imageName,
			imagePath:   getImagePath(k.Dir),
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
			volumeNamer:   namer.NewNamer(ctx, strings.TrimPrefix(strings.ReplaceAll(volKey, "/", "-"), "-")),
		}
		images = append(images, img)
		baseVolumeMap[k.Tag] = img
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        images,
		baseVolumeMap: baseVolumeMap,
		poolNamer:     namer.NewNamer(ctx, poolName),
	}, nil
}

func NewLibvirtFSCustomRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet) (*LibvirtFilesystem, error) {
	baseVolumeMap := make(map[string]*filesystemImage)
	poolName := vmset.Name
	imageName := vmset.Img.ImageName

	rc := resources.NewResourceCollection(vmset.Recipe)
	poolPath := generatePoolPath(poolName)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName:     pulumi.String(poolName),
			resources.PoolPath:     pulumi.String(poolPath),
			resources.User:         pulumi.String(currentUser.Uid),
			resources.LibvirtGroup: pulumi.String(libvirtGroup.Gid),
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
		volumeNamer:   namer.NewNamer(ctx, strings.TrimPrefix(strings.ReplaceAll(volKey, "/", "-"), "-")),
	}
	for _, k := range vmset.Kernels {
		baseVolumeMap[k.Tag] = img
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		poolName:      poolName,
		poolXML:       poolXML,
		poolXMLPath:   fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		images:        []*filesystemImage{img},
		baseVolumeMap: baseVolumeMap,
		poolNamer:     namer.NewNamer(ctx, poolName),
	}, nil
}

func downloadRootfs(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var downloadCmd string
	var waitFor []pulumi.Resource

	for _, fsImage := range fs.images {
		url, err := url.Parse(fsImage.imageSource)
		if err != nil {
			return []pulumi.Resource{}, fmt.Errorf("error parsing url %s: %w", fsImage.imageSource, err)
		}

		refreshCmd := fmt.Sprintf(refreshFromEBS, url.Path)
		if url.Scheme == "file" {
			// We do this because reading the EBS blocks is the only way to download the files
			// from the backing storage. Not doing this means, that the file is downloaded when
			// it is first accessed in other commands. This can cause other problems, on top of
			// being very slow.
			if url.Path != fsImage.imagePath {
				downloadCmd = fmt.Sprintf("%s && mv %s %s", refreshCmd, url.Path, fsImage.imagePath)
			} else {

				downloadCmd = refreshCmd
			}
		} else {
			downloadCmd = fmt.Sprintf("curl -o %s %s", fsImage.imagePath, fsImage.imageSource)
		}

		downloadRootfsArgs := command.Args{
			Create: pulumi.String(downloadCmd),
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

func extractRootfs(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, fsImage := range fs.images {
		// Extract archive if it is xz compressed, which will be the case when downloading from remote S3 bucket.
		// To do this we check if the magic bytes of the file at imagePath is 514649fb. If so then this is already
		// a qcow2 file. No need to extract it. Otherwise it SHOULD be xz archive. Attempt to uncompress.
		extractTopLevelArchive := command.Args{
			Create: pulumi.Sprintf("if xxd -p -l 4 %s | grep -E '^514649fb$'; then echo '%s is qcow2 file'; else mv %s %s.xz && xz -T0 -c -d %s.xz > %s; fi", fsImage.imagePath, fsImage.imagePath, fsImage.imagePath, fsImage.imagePath, fsImage.imagePath, fsImage.imagePath),
			Delete: pulumi.Sprintf("rm -rf %s", fsImage.imagePath),
		}
		res, err := runner.Command(fsImage.volumeNamer.ResourceName("extract-base-volume-package"), &extractTopLevelArchive, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, res)
	}

	return waitFor, nil
}

func setupLibvirtVMSetPool(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
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

func setupLibvirtVMVolume(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
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

func (fs *LibvirtFilesystem) SetupLibvirtFilesystem(provider *libvirt.Provider, runner *Runner, arch string, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	switch arch {
	case LocalVMSet:
		return setupLocalLibvirtFilesystem(fs, provider, depends)
	default:
		return setupRemoteLibvirtFilesystem(fs, runner, depends)
	}
}

func setupRemoteLibvirtFilesystem(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
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

func setupLocalLibvirtFilesystem(fs *LibvirtFilesystem, provider *libvirt.Provider, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	poolReady, err := libvirt.NewPool(fs.ctx, fs.poolName, &libvirt.PoolArgs{
		Type: pulumi.String("dir"),
		Path: pulumi.String(generatePoolPath(fs.poolName)),
		Xml: libvirt.PoolXmlArgs{
			Xslt: fs.poolXML,
		},
	}, pulumi.Provider(provider), pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}
	waitFor = append(waitFor, poolReady)

	for _, fsImage := range fs.images {
		stgvolReady, err := libvirt.NewVolume(fs.ctx, fsImage.volumeNamer.ResourceName("build-libvirt-basevolume"), &libvirt.VolumeArgs{
			Pool:   pulumi.String(fs.poolName),
			Source: pulumi.String(fsImage.imagePath),
			Xml: libvirt.VolumeXmlArgs{
				Xslt: fsImage.volumeXML,
			},
		}, pulumi.Provider(provider), pulumi.DependsOn(waitFor))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, stgvolReady)
	}

	return waitFor, nil
}
