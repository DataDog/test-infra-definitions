package hyperv

import (
	"github.com/DataDog/test-infra-definitions/components/vm"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VMArgs struct {
	Name string
	// Attributes you need when you will actually create the VM
}

func NewVM(e Environment, args VMArgs, opts ...pulumi.ResourceOption) (vm.VM, error) {
	cmd, err := local.NewCommand(e.Ctx, e.CommonNamer.ResourceName("hyperv", args.Name), &local.CommandArgs{
		Interpreter: pulumi.ToStringArray([]string{"powershell", "-Command"}),
		Environment: pulumi.StringMap{},                        // if you need to inject environment variables
		Create:      pulumi.String(`Write-Host "Hello World"`), // What to do when you create the resource. Creating the VM or reading some file to get the info
		Update:      pulumi.String(`Write-Host "Hello World"`), // What to do when you update the resource. If empty, `Create` will be used, usually nothing specific required.
		Delete:      pulumi.String(`Write-Host "Hello World"`), // What to do when you delete the resource. If empty, nothing will be done.
		Triggers:    pulumi.Array{},                            // If you need to trigger the resource creation/update/delete based on some other resource
		AssetPaths:  pulumi.StringArray{},                      // If you need to get file content from the local filesystem instead of reading stdout to get info
		Dir:         pulumi.String(""),                         // The directory to run the command in. Defaults to the Pulumi program's directory.
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Let's say you get IP address from the command output (only output in the command).
	os := newWindows(e)
	vm, err := vm.NewGenericVM(args.Name, cmd, &e, cmd.Stdout, os)
	if err != nil {
		return nil, err
	}

	return vm, nil
}
