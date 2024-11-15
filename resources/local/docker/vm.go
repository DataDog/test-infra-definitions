package localdocker

import (
	_ "embed"
	"os"
	"path"
	"runtime"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/utils"
	resourceslocal "github.com/DataDog/test-infra-definitions/resources/local"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VMArgs struct {
	Name string
}

//go:embed data/Dockerfile
var dockerfileContent string

func NewInstance(e resourceslocal.Environment, args VMArgs, opts ...pulumi.ResourceOption) (address pulumi.StringOutput, user string, port int, err error) {
	interpreter := []string{"/bin/bash", "-c"}
	if runtime.GOOS == "windows" {
		interpreter = []string{"powershell", "-Command"}
	}

	publicKey, err := os.ReadFile(e.DefaultPublicKeyPath())
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}
	dataPath := path.Join(homeDir, ".localdocker")
	// TODO clean up the folder on stack destroy
	// Requires a Runner refactor to reuse crossplatform commands
	err = os.MkdirAll(dataPath, 0700)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}
	dockerfilePath := path.Join(dataPath, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0600)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}

	opts = utils.MergeOptions(opts, e.WithProviders(config.ProviderCommand))
	// TODO use NewLocalRunner
	// requires a refactor to pass interpreter
	buildPodman, err := local.NewCommand(e.Ctx(), e.CommonNamer().ResourceName("podman-build", args.Name), &local.CommandArgs{
		Interpreter: pulumi.ToStringArray(interpreter),
		Environment: pulumi.StringMap{"DOCKER_HOST_SSH_PUBLIC_KEY": pulumi.String(string(publicKey))},
		Create:      pulumi.Sprintf("podman build --format=docker --build-arg DOCKER_HOST_SSH_PUBLIC_KEY=\"$DOCKER_HOST_SSH_PUBLIC_KEY\" -t %s .", args.Name),
		Delete:      pulumi.Sprintf("podman rmi %s", args.Name),
		Triggers:    pulumi.Array{},
		AssetPaths:  pulumi.StringArray{},
		Dir:         pulumi.String(dataPath),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}
	opts = utils.MergeOptions(opts, pulumi.DependsOn([]pulumi.Resource{buildPodman}))
	runPodman, err := local.NewCommand(e.Ctx(), e.CommonNamer().ResourceName("podman-run", args.Name), &local.CommandArgs{
		Interpreter: pulumi.ToStringArray(interpreter),
		Environment: pulumi.StringMap{"DOCKER_HOST_SSH_PUBLIC_KEY": pulumi.String(string(publicKey))},
		Create:      pulumi.Sprintf("podman run -d --name=%[1]s_run -p 50022:22 %[1]s", args.Name),
		Delete:      pulumi.Sprintf("podman stop %[1]s_run && podman rm %[1]s_run", args.Name),
		Triggers:    pulumi.Array{},
		AssetPaths:  pulumi.StringArray{},
		Dir:         pulumi.String(dataPath),
	}, opts...)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}

	e.Ctx().Log.Info("Running with container of type ubuntu", nil)

	// hack to wait for the container to be up
	ipAddress := runPodman.Stdout.ApplyT(func(_ string) string {
		return "localhost"
	}).(pulumi.StringOutput)

	return ipAddress, "root", 50022, nil
}
