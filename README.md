# Dynamic infrastructures for test

This repository contains IaC code based on Pulumi to provision dynamic test infrastructures for testing.

## Quick start guide

The first step is to download and install Pulumi CLI. Information can be found [here](https://www.pulumi.com/docs/get-started/install/).

For instance, on MacOS:

```bash
brew install pulumi/tap/pulumi
```

### Configure an AWS ssh key pair

1. Login to [https://console.aws.amazon.com/ec2/](the AWS EC2 console)
2. In the navigation pane, under Network & Security, choose Key Pairs.
3. On the Key Pairs page, click on Actions -> Import key pair.
4. Name the key pair with your Github handle and browse to `~/.ssh/id_rsa.pub`. You don't have it ? You can create an ssh key pair following `SSH Key Setup`.

### Stack & Storage

Pulumi requires to store/retrieve the state of your `Stack`.
In Pulumi, `Stack` objects represent your actual deployment:
- A `Stack` references a `Project` (a folder with a `Pulumi.yaml`, for instance `./aws/eks`)
- A `Stack` references a configuration file called `Pulumi.<stack_name>.yaml`
This file holds your `Stack` configuration.
If it does not exist, it will be created.
If it exists and you input some settings through the command line, using `-c`, it will update the `Stack` file.

When performing operations on a `Stack`, Pulumi will need to store a state somewhere (the Stack state).
Normally the state should be stored in a durable storage (e.g. S3-like), but for testing purposes
local filesystem could be used.

To choose a default storage provider, use `pulumi login` (should be only done once):

```bash
# Using local filesystem (state will be stored in ~/.pulumi)
pulumi login --local

# Using storage on Cloud Storage (GCP)
# You need to create the bucket on sandbox.
# You also need to have sandbox as your current tenant in gcloud CLI.
pulumi login gs://<your_name>-pulumi
```

More information about state can be retrieved at: https://www.pulumi.com/docs/intro/concepts/state/

Finally, Pulumi is encrypting secrets in your `Pulumi.<stack_name>.yaml` (if entered as such).
To do that, it requires a password. For dev purposes, you can simply store the password in the `PULUMI_CONFIG_PASSPHRASE` variable in your `~/.zshrc`.

### Create a stack

In this example, we're going to create an ECS Cluster:

```bash
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi up -c ddinfra:aws/defaultKeyPairName=<your_github_handle> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -C ./aws/ecs -s <your_github_handle>-ecs-test
```

In case of failure, you may update some parameters or configuration and run the command again.
Note that all `-c` parameters have been set in your `Pulumi.<stack_name>.yaml` file.

**NOTE:** Do not commit your Stack file.

### List existing stacks

To list the instances of a stack run

```bash
pulumi stack -C <path/to/stack> ls 

# List scenarios/ndm/snmp
pulumi stack -C ./scenarios/ndm/snmp ls
```

### Destroying a stack

Once you're finished with the test environment you've created, you can safely delete it.
To do this, we'll use the `destroy` operation referecing our `Stack` file:

```bash
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi destroy -C ./aws/ecs -s <your_name>-ecs-test
```

Note that we don't need to use `-c` again as the configuration values were put into the `Stack` file.
This will destroy all cloud resources associated to the Stack, but the state itself (mostly empty) will still be there.
To remove the stack state:

```bash
pulumi stack rm <your_github_handle>-ecs-test
```

## Quick start: A VM with Agent deployed (Host install)

```bash
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi up -c ddinfra:aws/defaultKeyPairName=<your_github_handle> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -c ddinfra:aws/defaultPrivateKeyPath=$HOME/.ssh/id_rsa  -C ./aws/scenarios/vm -s <your_github_handle>
```

## Quick start: A VM with Docker(/Compose) with Agent deployed

```bash
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi up -c ddinfra:aws/defaultKeyPairName=<your_github_handle> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -c ddinfra:aws/defaultPrivateKeyPath=$HOME/.ssh/id_rsa  -C ./aws/scenarios/docker-vm -s <your_github_handle>-docker
```

## Quick start: Create an ECS EC2 (Windows/Linux) + Fargate (Linux) Cluster

```bash
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi up -c ddinfra:aws/defaultKeyPairName=<your_github_handle> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -C ./aws/ecs -s <your_github_handle>-ecs
```

## Quick start: Create an EKS (Linux/Windows) + Fargate (Linux) Cluster + Agent (Helm)

```bash
# You need to have a DD APIKey AND APPKey in variable DD_API_KEY / DD_APP_KEY
pulumi up -c ddinfra:aws/defaultKeyPairName=<your_github_handle> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -c ddagent:appKey=$DD_APP_KEY -C ./aws/eks -s <your_github_handle>-eks
```
