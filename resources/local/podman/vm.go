package localpodman

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
var customDockerConfig = "{}"

func NewInstance(e resourceslocal.Environment, args VMArgs, opts ...pulumi.ResourceOption) (address pulumi.StringOutput, user string, port int, err error) {
	// TODO A: Unix command / windows
	// runner := command.NewLocalRunner(&e, command.LocalRunnerArgs{OSCommand: command.NewUnixOSCommand()})
	// fileManager := command.NewFileManager(runner)

	// runner.CopyUnixFile("copy-hey-ho", pulumi.String("/tmp/hey"), pulumi.String("/tmp/ho"))

	// runner.Command("hey-ho", &command.Args{
	// 	Create: pulumi.String("cp /tmp/hey /tmp/ho"),
	// 	Delete: pulumi.String("rm /tmp/ho"),
	// })
	// TODO END

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
	dataPath := path.Join(homeDir, ".localpodman")
	// TODO clean up the folder on stack destroy
	// Requires a Runner refactor to reuse crossplatform commands
	err = os.MkdirAll(dataPath, 0700)
	// _, err = fileManager.CreateDirectory(dataPath, false)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}
	println("DIR")
	dockerfilePath := path.Join(dataPath, "Dockerfile")
	// _, err = fileManager.CopyInlineFile(pulumi.String(dockerfileContent), dockerfilePath)
	err = os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0600)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}
	println("COPY")

	// Use a config to avoid docker hooks that can call vault or other services (credHelpers)
	err = os.WriteFile(path.Join(dataPath, "config.json"), []byte(customDockerConfig), 0600)
	if err != nil {
		return pulumi.StringOutput{}, "", -1, err
	}

	podmanCommand := "podman --config " + dataPath

	opts = utils.MergeOptions(opts, e.WithProviders(config.ProviderCommand))
	// TODO use NewLocalRunner
	// requires a refactor to pass interpreter
	buildPodman, err := local.NewCommand(e.Ctx(), e.CommonNamer().ResourceName("podman-build", args.Name), &local.CommandArgs{
		Interpreter: pulumi.ToStringArray(interpreter),
		Environment: pulumi.StringMap{"DOCKER_HOST_SSH_PUBLIC_KEY": pulumi.String(string(publicKey))},
		Create:      pulumi.Sprintf("%s build --format=docker --build-arg DOCKER_HOST_SSH_PUBLIC_KEY=\"$DOCKER_HOST_SSH_PUBLIC_KEY\" -t %s .", podmanCommand, args.Name),
		Delete:      pulumi.Sprintf("%s rmi %s", podmanCommand, args.Name),
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
		Create:      pulumi.Sprintf("%s run -d --name=%[2]s_run -p 50022:22 %[2]s", podmanCommand, args.Name),
		Delete:      pulumi.Sprintf("%s stop %[2]s_run && podman rm %[2]s_run", podmanCommand, args.Name),
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
