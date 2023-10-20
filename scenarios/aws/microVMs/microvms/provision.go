package microvms

import (
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/microVMs/microvms/resources"
)

const DockerMountpoint = "/mnt/docker"

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

func setupMicroVMSSHConfig(instance *Instance, microVMIPSubnet string, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	createSSHDirArgs := command.Args{
		Create: pulumi.Sprintf("mkdir -p /home/ubuntu/.ssh && chmod 700 /home/ubuntu/.ssh"),
	}
	createDirDone, err := instance.runner.Command(instance.instanceNamer.ResourceName("add-microvm-ssh-dir", microVMIPSubnet), &createSSHDirArgs, pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}

	pattern := getMicroVMGroupSubnetPattern(microVMIPSubnet)
	args := command.Args{
		Create: pulumi.Sprintf(`echo -e "Host %s\nIdentityFile /home/kernel-version-testing/ddvm_rsa\nUser root\nStrictHostKeyChecking no\n" | tee /home/ubuntu/.ssh/config && chmod 600 /home/ubuntu/.ssh/config`, pattern),
	}
	done, err := instance.runner.Command(instance.instanceNamer.ResourceName("add-microvm-ssh-config", microVMIPSubnet), &args, pulumi.DependsOn([]pulumi.Resource{createDirDone}))
	if err != nil {
		return nil, err
	}
	return []pulumi.Resource{done}, nil
}

func readMicroVMSSHKey(instance *Instance, depends []pulumi.Resource) (pulumi.StringOutput, []pulumi.Resource, error) {
	args := command.Args{
		Create: pulumi.Sprintf("cat /home/kernel-version-testing/ddvm_rsa"),
	}
	done, err := instance.runner.RemoteCommand(instance.instanceNamer.ResourceName("read-microvm-ssh-key"), &args, pulumi.DependsOn(depends))
	if err != nil {
		return pulumi.StringOutput{}, nil, err
	}
	s := pulumi.ToSecret(done.Stdout).(pulumi.StringOutput)
	return s, []pulumi.Resource{done}, err
}

func setupSSHAllowEnv(runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	args := command.Args{
		Create: pulumi.Sprintf("echo -e 'AcceptEnv DD_API_KEY\n' | sudo tee -a /etc/ssh/sshd_config"),
	}
	done, err := runner.Command("allow-ssh-env", &args, pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}
	return []pulumi.Resource{done}, nil
}

func reloadSSHD(runner *Runner, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	args := command.Args{
		Create: pulumi.Sprintf("sudo systemctl reload sshd.service"),
	}
	done, err := runner.Command("reload sshd", &args, pulumi.DependsOn(depends))
	if err != nil {
		return nil, err
	}
	return []pulumi.Resource{done}, nil
}

func mountMicroVMDisks(runner *Runner, disks []resources.DomainDisk, namer namer.Namer, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, d := range disks {
		if d.Mountpoint == RootMountpoint {
			continue
		}

		args := command.Args{
			Create: pulumi.Sprintf("mkdir %[1]s && mount %[2]s %[1]s", d.Mountpoint, d.Target),
		}

		done, err := runner.Command(namer.ResourceName("mount-disk", fsPathToLibvirtResource(d.Target)), &args, pulumi.DependsOn(depends))
		if err != nil {
			return nil, err
		}

		waitFor = append(waitFor, done)
	}

	return waitFor, nil
}

func setDockerDataRoot(runner *Runner, disks []resources.DomainDisk, namer namer.Namer, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	var waitFor []pulumi.Resource

	for _, d := range disks {
		if d.Mountpoint != DockerMountpoint {
			continue
		}

		args := command.Args{
			Create: pulumi.Sprintf("sh -c 'systemctl stop docker && echo '{}' | jq -n '. += {\"data-root\":\"/mnt/docker\"}' > /etc/docker/daemon.json && systemctl start docker'", d.Mountpoint),
			Sudo:   true,
		}
		done, err := runner.Command(namer.ResourceName("set-docker-data-root"), &args, pulumi.DependsOn(depends))
		if err != nil {
			return nil, err
		}

		waitFor = append(waitFor, done)

		break
	}

	return waitFor, nil
}

// This function provisions the metal instance for setting up libvirt based micro-vms.
func provisionMetalInstance(instance *Instance) ([]pulumi.Resource, error) {
	if instance.Arch == LocalVMSet {
		return nil, nil
	}

	allowEnvDone, err := setupSSHAllowEnv(instance.runner, nil)
	if err != nil {
		return nil, err
	}

	reloadSSHDDone, err := reloadSSHD(instance.runner, allowEnvDone)
	if err != nil {
		return nil, err
	}
	return reloadSSHDDone, nil
}

func prepareLibvirtSSHKeys(runner *Runner, localRunner *command.LocalRunner, resourceNamer namer.Namer, pair sshKeyPair, depends []pulumi.Resource) ([]pulumi.Resource, error) {
	sshGenArgs := command.Args{
		Create: pulumi.Sprintf("rm -f %s && rm -f %s && ssh-keygen -t rsa -b 4096 -f %s -q -N \"\" && cat %s", pair.privateKey, pair.publicKey, pair.privateKey, pair.publicKey),
		Delete: pulumi.Sprintf("rm %s && rm %s", pair.privateKey, pair.publicKey),
	}
	sshgenDone, err := localRunner.Command(resourceNamer.ResourceName("gen-libvirt-sshkey"), &sshGenArgs)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return []pulumi.Resource{sshWrite}, nil
}
