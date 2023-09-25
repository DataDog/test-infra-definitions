package microvms

import (
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/DataDog/test-infra-definitions/components/command"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// This function provisions the metal instance for setting up libvirt based micro-vms.
func provisionInstance(instance *Instance) ([]pulumi.Resource, error) {
	return []pulumi.Resource{}, nil
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
