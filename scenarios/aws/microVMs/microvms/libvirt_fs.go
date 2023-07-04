package microvms

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/vmconfig"
	"github.com/pulumi/pulumi-libvirt/sdk/go/libvirt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

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

type LibvirtPool struct {
	poolName    string
	poolXML     pulumi.StringOutput
	poolXMLPath string
	poolNamer   namer.Namer
}

type LibvirtFilesystem struct {
	ctx           *pulumi.Context
	pool          *LibvirtPool
	images        []*filesystemImage
	baseVolumeMap map[string]*filesystemImage
	isLocal       bool
}

func generatePoolPath(name string) string {
	return fmt.Sprintf("/home/kernel-version-testing/libvirt/pools/%s", name)
}

func generateVolumeKey(pool, volName string) string {
	return fmt.Sprintf("%s/%s", generatePoolPath(pool), volName)
}

func rootFSDir() string {
	return filepath.Join(GetWorkingDirectory(), "rootfs")
}

func getImagePath(name string) string {
	return filepath.Join(rootFSDir(), name)
}

func fsPathToLibvirtResource(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "/", "-"), "-")
}

func NewLibvirtPool(ctx *pulumi.Context) *LibvirtPool {
	rc := resources.NewResourceCollection(vmconfig.RecipeDefault)
	poolName := libvirtResourceName(ctx.Stack(), "global-pool")
	poolPath := generatePoolPath(poolName)
	poolXML := rc.GetPoolXML(
		map[string]pulumi.StringInput{
			resources.PoolName: pulumi.String(poolName),
			resources.PoolPath: pulumi.String(poolPath),
		},
	)
	return &LibvirtPool{
		poolName:    poolName,
		poolXML:     poolXML,
		poolXMLPath: fmt.Sprintf("/tmp/pool-%s.tmp", poolName),
		poolNamer:   libvirtResourceNamer(ctx, poolName),
	}
}

func NewLibvirtFSDistroRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet, pool *LibvirtPool) *LibvirtFilesystem {
	var images []*filesystemImage

	rc := resources.NewResourceCollection(vmset.Recipe)
	baseVolumeMap := make(map[string]*filesystemImage)

	for _, k := range vmset.Kernels {
		imageName := pool.poolName + "-" + k.Tag
		volKey := generateVolumeKey(pool.poolName, imageName)
		img := &filesystemImage{
			imageName:   imageName,
			imagePath:   getImagePath(k.Dir),
			imageSource: k.ImageSource,
			volumeKey:   volKey,
			volumeXML: rc.GetVolumeXML(
				map[string]pulumi.StringInput{
					resources.ImageName: pulumi.String(imageName),
					resources.VolumeKey: pulumi.String(volKey),
					resources.ImagePath: pulumi.String(getImagePath(k.Dir)),
				},
			),
			volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
			// libvirt complains when volume name contains '/'. We replace with '-'
			volumeNamer: libvirtResourceNamer(ctx, fsPathToLibvirtResource(volKey)),
		}
		images = append(images, img)
		baseVolumeMap[k.Tag] = img
	}

	local := false
	if vmset.Arch == LocalVMSet {
		local = true
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		pool:          pool,
		images:        images,
		baseVolumeMap: baseVolumeMap,
		isLocal:       local,
	}
}

func NewLibvirtFSCustomRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet, pool *LibvirtPool) *LibvirtFilesystem {
	baseVolumeMap := make(map[string]*filesystemImage)
	imageName := vmset.Img.ImageName

	rc := resources.NewResourceCollection(vmset.Recipe)
	volKey := generateVolumeKey(pool.poolName, imageName)

	img := &filesystemImage{
		imageName:   imageName,
		imagePath:   getImagePath(imageName),
		imageSource: vmset.Img.ImageSourceURI,
		volumeKey:   volKey,
		volumeXML: rc.GetVolumeXML(
			map[string]pulumi.StringInput{
				resources.ImageName: pulumi.String(imageName),
				resources.VolumeKey: pulumi.String(volKey),
				resources.ImagePath: pulumi.String(getImagePath(imageName)),
			},
		),
		volumeXMLPath: fmt.Sprintf("/tmp/volume-%s.xml", imageName),
		// libvirt complains when volume name contains '/'. We replace with '-'
		volumeNamer: libvirtResourceNamer(ctx, fsPathToLibvirtResource(volKey)),
	}
	for _, k := range vmset.Kernels {
		baseVolumeMap[k.Tag] = img
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		images:        []*filesystemImage{img},
		baseVolumeMap: baseVolumeMap,
		pool:          pool,
	}
}

func buildAria2ConfigEntry(sb *strings.Builder, source string, imagePath string) {
	dir := filepath.Dir(imagePath)
	out := filepath.Base(imagePath)
	fmt.Fprintf(sb, "%s\n", source)
	fmt.Fprintf(sb, " dir=%s\n", dir)
	fmt.Fprintf(sb, " out=%s\n", out)
}

func refreshFromBackingStore(fsImage *filesystemImage, runner *Runner, urlPath string, isLocal bool, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var downloadCmd string
	var refreshCmd string

	if isLocal {
		// For local environment we do not need to "download" the image from
		// a backing store.
		refreshCmd = "true"
	} else {
		refreshCmd = fmt.Sprintf(refreshFromEBS, urlPath)
	}
	// We do this because reading the EBS blocks is the only way to download the files
	// from the backing storage. Not doing this means, that the file is downloaded when
	// it is first accessed in other commands. This can cause other problems, on top of
	// being very slow.
	if urlPath != fsImage.imagePath {
		downloadCmd = fmt.Sprintf("%s && mv %s %s", refreshCmd, urlPath, fsImage.imagePath)
	} else {

		downloadCmd = refreshCmd
	}
	downloadRootfsArgs := command.Args{
		Create: pulumi.String(downloadCmd),
		Delete: pulumi.Sprintf("rm -f %s", fsImage.imagePath),
	}

	res, err := runner.Command(fsImage.volumeNamer.ResourceName("download-rootfs"), &downloadRootfsArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{res}, err
}

func downloadRootfs(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var aria2DownloadConfig strings.Builder

	webDownload := false
	for _, fsImage := range fs.images {
		url, err := url.Parse(fsImage.imageSource)
		if err != nil {
			return []pulumi.Resource{}, fmt.Errorf("error parsing url %s: %w", fsImage.imageSource, err)
		}

		if url.Scheme == "file" {
			resources, err := refreshFromBackingStore(fsImage, runner, url.Path, fs.isLocal, depends)
			if err != nil {
				return waitFor, err
			}

			waitFor = append(waitFor, resources...)
		} else {
			buildAria2ConfigEntry(&aria2DownloadConfig, fsImage.imageSource, fsImage.imagePath)
			webDownload = true
		}
	}

	if webDownload {
		writeConfigFile := command.Args{
			Create: pulumi.Sprintf("echo \"%s\" > /tmp/aria2.config", aria2DownloadConfig.String()),
		}
		writeConfigFileDone, err := runner.Command(fs.pool.poolNamer.ResourceName("write-aria2c-config"), &writeConfigFile)
		if err != nil {
			return waitFor, err
		}
		downloadWithAria2Args := command.Args{
			Create: pulumi.String("aria2c -i /tmp/aria2.config -x 16 -j $(cat /tmp/aria2.config | grep dir | wc -l)"),
		}

		depends = append(depends, writeConfigFileDone)
		downloadWithAria2Done, err := runner.Command(fs.pool.poolNamer.ResourceName("download-with-aria2c"), &downloadWithAria2Args, pulumi.DependsOn(depends))
		if err != nil {
			return waitFor, err
		}

		waitFor = append(waitFor, downloadWithAria2Done)
	}

	return waitFor, nil
}

func setupLibvirtVMSetPool(pool *LibvirtPool, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolBuildReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-build %s", pool.poolName),
		Delete: pulumi.Sprintf("virsh pool-delete %s", pool.poolName),
		Sudo:   true,
	}
	poolStartReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-start %s", pool.poolName),
		Delete: pulumi.Sprintf("virsh pool-destroy %s", pool.poolName),
		Sudo:   true,
	}
	poolRefreshDoneArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-refresh %s", pool.poolName),
		Sudo:   true,
	}

	poolDefineReadyArgs := command.Args{
		Create: pulumi.Sprintf("virsh pool-define %s", pool.poolXMLPath),
		Sudo:   true,
	}

	poolDefineReady, err := runner.Command(pool.poolNamer.ResourceName("define-libvirt-pool"), &poolDefineReadyArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolBuildReady, err := runner.Command(pool.poolNamer.ResourceName("build-libvirt-pool"), &poolBuildReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolDefineReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolStartReady, err := runner.Command(pool.poolNamer.ResourceName("start-libvirt-pool"), &poolStartReadyArgs, pulumi.DependsOn([]pulumi.Resource{poolBuildReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	poolRefreshDone, err := runner.Command(pool.poolNamer.ResourceName("refresh-libvirt-pool"), &poolRefreshDoneArgs, pulumi.DependsOn([]pulumi.Resource{poolStartReady}))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{poolRefreshDone}, err
}

func setupLibvirtVMVolume(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, fsImage := range fs.images {
		baseVolumeReadyArgs := command.Args{
			Create: pulumi.Sprintf("virsh vol-create %s %s", fs.pool.poolName, fsImage.volumeXMLPath),
			Delete: pulumi.Sprintf("virsh vol-delete %s --pool %s", fsImage.volumeKey, fs.pool.poolName),
			Sudo:   true,
		}

		baseVolumeReady, err := runner.Command(fsImage.volumeNamer.ResourceName("build-libvirt-basevolume"), &baseVolumeReadyArgs, pulumi.DependsOn(depends))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, baseVolumeReady)
	}

	return waitFor, nil
}

func (fs *LibvirtFilesystem) SetupLibvirtFilesystem(provider *libvirt.Provider, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	if fs.isLocal {
		return setupLocalLibvirtFilesystem(fs, provider, depends)
	}

	return setupRemoteLibvirtFilesystem(fs, runner, depends)
}

func setupRemoteLibvirtPool(pool *LibvirtPool, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	poolXMLWrittenArgs := command.Args{
		Create: pulumi.Sprintf("echo \"%s\" > %s", pool.poolXML, pool.poolXMLPath),
		Delete: pulumi.Sprintf("rm -f %s", pool.poolXMLPath),
	}
	poolXMLWritten, err := runner.Command(pool.poolNamer.ResourceName("write-pool-xml"), &poolXMLWrittenArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	setupLibvirtVMPoolDone, err := setupLibvirtVMSetPool(pool, runner, []pulumi.Resource{poolXMLWritten})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return setupLibvirtVMPoolDone, err
}

// poolDone Resoures are passed separately to optimize the order in which the filesystem setup is done.
// This is the slowest part of the process, and the downloading of the images should begin as early as possible.
// Therefore, we do not slow it down by waiting for the pool to become ready first.
func setupRemoteLibvirtFilesystem(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	downloadRootfsDone, err := downloadRootfs(fs, runner, depends)
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

	setupLibvirtVMVolumeDone, err := setupLibvirtVMVolume(fs, runner, append(volumeXMLReady, downloadRootfsDone...))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return setupLibvirtVMVolumeDone, nil
}

func setupLocalLibvirtFilesystem(fs *LibvirtFilesystem, provider *libvirt.Provider, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	poolReady, err := libvirt.NewPool(fs.ctx, fs.pool.poolNamer.ResourceName("create-libvirt-pool"), &libvirt.PoolArgs{
		Type: pulumi.String("dir"),
		Name: pulumi.String(fs.pool.poolName),
		Path: pulumi.String(generatePoolPath(fs.pool.poolName)),
	}, pulumi.Provider(provider), pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}
	waitFor = append(waitFor, poolReady)

	for _, fsImage := range fs.images {
		stgvolReady, err := libvirt.NewVolume(fs.ctx, fsImage.volumeNamer.ResourceName("build-libvirt-basevolume"), &libvirt.VolumeArgs{
			Name:   pulumi.String(fsImage.imageName),
			Pool:   pulumi.String(fs.pool.poolName),
			Source: pulumi.String(fsImage.imagePath),
			Xml: libvirt.VolumeXmlArgs{
				Xslt: fsImage.volumeXML,
			},
		}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{poolReady}))
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, stgvolReady)
	}

	return waitFor, nil
}
