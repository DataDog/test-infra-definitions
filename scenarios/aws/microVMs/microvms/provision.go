package microvms

import (
	"sync"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const sharedDiskCmd = `MYUSER=$(id -u) MYGROUP=$(id -g) sh -c \
'mkdir -p %[1]s/kmt-ramfs && \
sudo -E -S mount -t ramfs -o size=5g,uid=$MYUSER,gid=$MYGROUP,othmask=0077,mode=0777 ramfs %[1]s/kmt-ramfs && \
mkdir %[1]s/kmt-ramfs/deps && \
dd if=/dev/zero of=%[1]s/kmt-ramfs/deps.img bs=1G count=3 && \
mkfs.ext4 -F %[1]s/kmt-ramfs/deps.img && \
sudo -S mount -o exec,loop %[1]s/kmt-ramfs/deps.img %[1]s/kmt-ramfs/deps' \
`

var initSudoPassword sync.Once
var SudoPasswordLocal pulumi.StringOutput
var SudoPasswordRemote pulumi.StringOutput

func GetSudoPassword(ctx *pulumi.Context, isLocal bool) pulumi.StringOutput {
	initSudoPassword.Do(func() {
		rootConfig := config.New(ctx, "")
		SudoPasswordLocal = rootConfig.RequireSecret("sudo-password-local")
		SudoPasswordRemote = rootConfig.RequireSecret("sudo-password-remote")
	})

	if isLocal {
		return SudoPasswordLocal
	}

	return SudoPasswordRemote
}

func setupSharedDisk(runner *Runner, ctx *pulumi.Context, isLocal bool, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	buildSharedDiskInRamfsArgs := command.Args{
		Create: pulumi.Sprintf(sharedDiskCmd, GetWorkingDirectory()),
		Delete: pulumi.Sprintf("umount %[1]s/kmt-ramfs/deps && umount %[1]s/kmt-ramfs && rm -r %[1]s/kmt-ramfs", GetWorkingDirectory()),
		Stdin:  GetSudoPassword(ctx, isLocal),
	}

	buildSharedDiskInRamfsDone, err := runner.Command("build-shared-disk", &buildSharedDiskInRamfsArgs, pulumi.DependsOn(depends))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{buildSharedDiskInRamfsDone}, nil
}

// This function provisions the metal instance for setting up libvirt based micro-vms.
func provisionInstance(instance *Instance) ([]pulumi.Resource, error) {
	runner := instance.runner

	return setupSharedDisk(runner, instance.e.Ctx, instance.Arch == LocalVMSet, []pulumi.Resource{})
}

func prepareLibvirtSSHKeys(runner *Runner, localRunner *command.LocalRunner, resourceNamer namer.Namer, pair sshKeyPair, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	sshGenArgs := command.Args{
		Create: pulumi.Sprintf("rm -f %s && rm -f %s && ssh-keygen -t rsa -b 4096 -f %s -q -N \"\" && cat %s", pair.privateKey, pair.publicKey, pair.privateKey, pair.publicKey),
		Delete: pulumi.Sprintf("rm %s && rm %s", pair.privateKey, pair.publicKey),
	}
	sshgenDone, err := localRunner.Command(resourceNamer.ResourceName("gen-libvirt-sshkey"), &sshGenArgs)
	if err != nil {
		return []pulumi.Resource{}, err
	}

	// This command writes the public ssh key which pulumi uses to talk to the libvirt daemon, in the authorized_keys
	// file of the default user. We must write in this file because pulumi runs its commands as the default user.
	//
	// We override the runner-level user here with root, and construct the path to the default users .ssh directory,
	// in order to write the public ssh key in the correct file.
	sshWriteArgs := command.Args{
		Create: pulumi.Sprintf("echo '%s' >> $(getent passwd 1000 | cut -d: -f6)/.ssh/authorized_keys", sshgenDone.Stdout),
		Sudo:   true,
	}

	wait := append(depends, sshgenDone)
	sshWrite, err := runner.Command("write-ssh-key", &sshWriteArgs, pulumi.DependsOn(wait))
	if err != nil {
		return []pulumi.Resource{}, err
	}

	return []pulumi.Resource{sshWrite}, nil
}
