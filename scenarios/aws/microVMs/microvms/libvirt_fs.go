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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const refreshFromEBS = "fio --filename=%s --rw=read --bs=64m --iodepth=32 --ioengine=libaio --direct=1 --name=volume-initialize"

type LibvirtFilesystem struct {
	ctx           *pulumi.Context
	pools         map[vmconfig.PoolType]LibvirtPool
	volumes       []LibvirtVolume
	baseVolumeMap map[string][]LibvirtVolume
	fsNamer       namer.Namer
	isLocal       bool
}

func rootFSDir() string {
	return filepath.Join(GetWorkingDirectory(), "rootfs")
}

// libvirt complains when volume name contains '/'. We replace with '-'
func fsPathToLibvirtResource(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "/", "-"), "-")
}

func getNamer(ctx *pulumi.Context) func(string) namer.Namer {
	return func(volKey string) namer.Namer {
		return libvirtResourceNamer(ctx, fsPathToLibvirtResource(volKey))
	}
}

func buildVolumeResourceXMLFn(base map[string]pulumi.StringInput, recipe string) func(volumeKey string) pulumi.StringOutput {
	rc := resources.NewResourceCollection(recipe)
	return func(volumeKey string) pulumi.StringOutput {
		base[resources.VolumeKey] = pulumi.String(volumeKey)
		return rc.GetVolumeXML(base)
	}
}

func getImagePath(base, name string) string {
	return filepath.Join(base, name)
}

// vms created with the distro recipe can have different backing filesystem images for different VMs.
// For example ubuntu and fedora VMs would have different backing images.
func NewLibvirtFSDistroRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet, pools map[vmconfig.PoolType]LibvirtPool) *LibvirtFilesystem {
	var volumes []LibvirtVolume

	baseVolumeMap := make(map[string][]LibvirtVolume)
	defaultPool := pools[DefaultPool]
	for _, k := range vmset.Kernels {
		imageName := defaultPool.Name() + "-" + k.Tag
		vol := NewLibvirtVolume(
			defaultPool,
			filesystemImage{
				imageName:   imageName,
				imagePath:   getImagePath(filepath.Join(GetWorkingDirectory(), "rootfs"), k.Dir),
				imageSource: k.ImageSource,
			},
			buildVolumeResourceXMLFn(
				map[string]pulumi.StringInput{
					resources.ImageName: pulumi.String(imageName),
					resources.ImagePath: pulumi.String(getImagePath(filepath.Join(GetWorkingDirectory(), "rootfs"), k.Dir)),
				},
				vmset.Recipe,
			),
			getNamer(ctx),
		)
		volumes = append(volumes, vol)
		baseVolumeMap[k.Tag] = append(baseVolumeMap[k.Tag], vol)
	}

	for _, d := range vmset.Disks {
		imageName := d.ImageName
		storePath := strings.TrimPrefix(d.BackingStore, "file://")
		vol := NewLibvirtVolume(
			pools[d.Type],
			filesystemImage{
				imageName:   imageName,
				imagePath:   getImagePath(storePath, imageName),
				imageSource: d.BackingStore,
			},
			buildVolumeResourceXMLFn(
				map[string]pulumi.StringInput{
					resources.ImageName: pulumi.String(imageName),
					resources.ImagePath: pulumi.String(getImagePath(storePath, imageName)),
				},
				vmset.Recipe,
			),
			getNamer(ctx),
		)

		// associate extra disks with all vms
		for _, k := range vmset.Kernels {
			baseVolumeMap[k.Tag] = append(baseVolumeMap[k.Tag], vol)
		}

		volumes = append(volumes, vol)
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		pools:         pools,
		volumes:       volumes,
		baseVolumeMap: baseVolumeMap,
		fsNamer:       libvirtResourceNamer(ctx, vmset.Name),
		isLocal:       vmset.Arch == LocalVMSet,
	}
}

// vms created with the custom recipe all share the same debian based backing filesystem image.
func NewLibvirtFSCustomRecipe(ctx *pulumi.Context, vmset *vmconfig.VMSet, pools map[vmconfig.PoolType]LibvirtPool) *LibvirtFilesystem {
	var volumes []LibvirtVolume

	baseVolumeMap := make(map[string][]LibvirtVolume)
	imageName := vmset.Img.ImageName
	vol := NewLibvirtVolume(
		pools[DefaultPool],
		filesystemImage{
			imageName:   imageName,
			imagePath:   getImagePath(filepath.Join(GetWorkingDirectory(), "rootfs"), imageName),
			imageSource: vmset.Img.ImageSourceURI,
		},
		buildVolumeResourceXMLFn(
			map[string]pulumi.StringInput{
				resources.ImageName: pulumi.String(imageName),
				resources.ImagePath: pulumi.String(getImagePath(filepath.Join(GetWorkingDirectory(), "rootfs"), imageName)),
			},
			vmset.Recipe,
		),
		getNamer(ctx),
	)
	volumes = append(volumes, vol)

	for _, k := range vmset.Kernels {
		baseVolumeMap[k.Tag] = append(baseVolumeMap[k.Tag], vol)
	}

	for _, d := range vmset.Disks {
		imageName := pools[d.Type].Name() + "-" + d.ImageName
		storePath := strings.TrimPrefix(d.BackingStore, "file://")
		vol := NewLibvirtVolume(
			pools[d.Type],
			filesystemImage{
				imageName:   imageName,
				imagePath:   getImagePath(storePath, imageName),
				imageSource: d.BackingStore,
			},
			buildVolumeResourceXMLFn(
				map[string]pulumi.StringInput{
					resources.ImageName: pulumi.String(imageName),
					resources.ImagePath: pulumi.String(getImagePath(storePath, imageName)),
				},
				vmset.Recipe,
			),
			getNamer(ctx),
		)

		// associate extra disks with all vms
		for _, k := range vmset.Kernels {
			baseVolumeMap[k.Tag] = append(baseVolumeMap[k.Tag], vol)
		}

		volumes = append(volumes, vol)
	}

	return &LibvirtFilesystem{
		ctx:           ctx,
		volumes:       volumes,
		baseVolumeMap: baseVolumeMap,
		pools:         pools,
		fsNamer:       libvirtResourceNamer(ctx, vmset.Name),
		isLocal:       vmset.Arch == LocalVMSet,
	}
}

func buildAria2ConfigEntry(sb *strings.Builder, source string, imagePath string) {
	dir := filepath.Dir(imagePath)
	out := filepath.Base(imagePath)
	fmt.Fprintf(sb, "%s\n", source)
	fmt.Fprintf(sb, " dir=%s\n", dir)
	fmt.Fprintf(sb, " out=%s\n", out)
}

func refreshFromBackingStore(volume LibvirtVolume, runner *Runner, urlPath string, isLocal bool, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var downloadCmd string
	var refreshCmd string

	fsImage := volume.UnderlyingImage()
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

	res, err := runner.Command(volume.FullResourceName("download-rootfs"), &downloadRootfsArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{res}, err
}

func downloadRootfs(fs *LibvirtFilesystem, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	var aria2DownloadConfig strings.Builder

	webDownload := false
	for _, volume := range fs.volumes {
		// only download backing stores for volumes inside default pool since these are
		// the iamges from which VMs boot
		//
		// ignore other volume types since they are created by this scenario and not downloaded.
		if volume.Pool().Type() != DefaultPool {
			continue
		}

		fsImage := volume.UnderlyingImage()
		url, err := url.Parse(fsImage.imageSource)
		if err != nil {
			return []pulumi.Resource{}, fmt.Errorf("error parsing url %s: %w", fsImage.imageSource, err)
		}

		if url.Scheme == "file" {
			resources, err := refreshFromBackingStore(volume, runner, url.Path, fs.isLocal, depends)
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
		configPath := fmt.Sprintf("/tmp/aria2-%s.config", fs.fsNamer.ResourceName("aria2c"))
		writeConfigFile := command.Args{
			Create: pulumi.Sprintf("echo \"%s\" > %s", aria2DownloadConfig.String(), configPath),
			Update: pulumi.Sprintf("echo \"%s\" > %s", aria2DownloadConfig.String(), configPath),
		}
		writeConfigFileDone, err := runner.Command(fs.fsNamer.ResourceName("write-aria2c-config"), &writeConfigFile)
		if err != nil {
			return waitFor, err
		}

		// We allow this command to fail.
		// The '--auto-file-renaming' flag allows us to skip downloading already downloaded files.
		// However, it causes aria2c to fail if a control file for the corresponding file does not exist.
		// We let the update fail assuming that most of the failures are due to the above case. If there is
		// a problem downloading the file for some other reason, subsequent commands will fail, thus alerting us.
		downloadWithAria2Args := command.Args{
			Create:   pulumi.Sprintf("aria2c --auto-file-renaming=false -i %s -x 16 -j $(cat %s | grep dir | wc -l) || true", configPath, configPath),
			Triggers: pulumi.Array{pulumi.String(aria2DownloadConfig.String())},
			Stdin:    pulumi.String(aria2DownloadConfig.String()),
		}

		depends = append(depends, writeConfigFileDone)
		downloadWithAria2Done, _ := runner.Command(fs.fsNamer.ResourceName("download-with-aria2c"), &downloadWithAria2Args, pulumi.DependsOn(depends))

		waitFor = append(waitFor, downloadWithAria2Done)
	}

	return waitFor, nil
}

func (fs *LibvirtFilesystem) SetupLibvirtFilesystem(providerFn LibvirtProviderFn, runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	// Downloading the base images for the volumes is the slowest part of the entire setup.
	// We want this step to start as soon as our remote VMs are ready. Therefore, we do not
	// make it depend on any other step.
	//
	// [IMPORTANT] The download may start as the first step. So if the setup changes such that the download
	// becomes dependent on some prior step, this call should change !!
	downloadRootfsDone, err := downloadRootfs(fs, runner, []pulumi.Resource{})
	if err != nil {
		return []pulumi.Resource{}, err
	}

	depends = append(depends, downloadRootfsDone...)
	return setupLibvirtFilesystem(fs, runner, providerFn, depends)
}

func setupLibvirtFilesystem(fs *LibvirtFilesystem, runner *Runner, providerFn LibvirtProviderFn, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource
	for _, vol := range fs.volumes {
		setupLibvirtVMVolumeDone, err := vol.SetupLibvirtVMVolume(fs.ctx, runner, providerFn, fs.isLocal, depends)
		if err != nil {
			return []pulumi.Resource{}, err
		}

		waitFor = append(waitFor, setupLibvirtVMVolumeDone)
	}

	return waitFor, nil
}
