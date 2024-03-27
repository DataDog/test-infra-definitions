# Dynamic infrastructures for test

This repository contains IaC code based on Pulumi to provision dynamic test infrastructures for testing.

## Prerequisites

To run scripts and code in this repository, you will need:

* [Go](https://golang.org/doc/install) 1.19 or later. You'll also need to set your `$GOPATH` and have `$GOPATH/bin` in your path.
* Python 3.7+ along with development libraries for tooling.
* `account-admin` role on AWS `agent-sandbox` account. Ensure it by running
  
  ```bash
  aws-vault login sso-agent-sandbox-account-admin
  ```

This guide is tested on **MacOS**.

##### List of Linux dependencies

```bash
sudo apt install libnotify-bin
```

## Quick start guide

1. Clone this repository

```bash
cd ~/dd && git clone git@github.com:DataDog/test-infra-definitions.git 
```

2. Install Python dependencies

```bash
cd ~/dd/test-infra-definitions && pip3 install --requirement requirements.txt
```

3. Add a PULUMI_CONFIG_PASSPHRASE to your Terminal rc file. Create a random password using 1Password and store it there

```bash
export PULUMI_CONFIG_PASSPHRASE=<random password stored in 1Password>
```

4. Run and follow the setup script

```bash
inv setup
```

### Create an environment for manual tests


Invoke tasks help deploying most common environments - VMs, Docker, ECS, EKS. Run `inv -l` to learn more.

```bash
inv -l

Available tasks:

  create-aks       Create a new AKS environment. It lasts around 5 minutes.
  create-docker    Create a docker environment.
  create-ecs       Create a new ECS environment.
  create-eks       Create a new EKS environment. It lasts around 20 minutes.
  create-vm        Create a new virtual machine on the cloud.
  destroy-aks      Destroy a AKS environment created with invoke create-aks.
  destroy-docker   Destroy an environment created by invoke create_docker.
  destroy-ecs      Destroy a ECS environment created with invoke create-ecs.
  destroy-eks      Destroy a EKS environment created with invoke create-eks.
```

Run any `-h` on any of the available tasks for more information

### Pulumi: Stack & Storage

Pulumi requires to store/retrieve the state of your `Stack`.
In Pulumi, `Stack` objects represent your actual deployment:
- A `Stack` references a `Project` (a folder with a `Pulumi.yaml`, for instance root folder of this repo)
- A `Stack` references a configuration file called `Pulumi.<stack_name>.yaml`
This file holds your `Stack` configuration.
If it does not exist, it will be created.
If it exists and you input some settings through the command line, using `-c`, it will update the `Stack` file.

When performing operations on a `Stack`, Pulumi will need to store a state somewhere (the Stack state).
Normally the state should be stored in a durable storage (e.g. S3-like), but for testing purposes
local filesystem could be used.

To choose a default storage provider, use `pulumi login` (should be only done once):

```
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

### Creating a stack with `pulumi up`

In this example, we're going to create an ECS Cluster:

```
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sso-agent-sandbox-account-admin -- pulumi up -c scenario=aws/ecs -c ddinfra:aws/defaultKeyPairName=<your_exisiting_aws_keypair_name> -c ddinfra:env=aws/agent-sandbox -c ddagent:apiKey=$DD_API_KEY -s <your_name>-ecs-test
```

In case of failure, you may update some parameters or configuration and run the command again.
Note that all `-c` parameters have been set in your `Pulumi.<stack_name>.yaml` file.

**NOTE:** Do not commit your Stack file.

### Destroying a stack

Once you're finished with the test environment you've created, you can safely delete it.
To do this, we'll use the `destroy` operation referecing our `Stack` file:

```
aws-vault exec sso-agent-sandbox-account-admin -- pulumi destroy -s <your_name>-ecs-test
```

Note that we don't need to use `-c` again as the configuration values were put into the `Stack` file.
This will destroy all cloud resources associated to the Stack, but the state itself (mostly empty) will still be there.
To remove the stack state:

```
pulumi stack rm <your_name>-ecs-test
```

## Quick start: A VM with Docker(/Compose) with Agent deployed

```
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sso-agent-sandbox-account-admin -- pulumi up -c scenario=aws/dockervm -c ddinfra:aws/defaultKeyPairName=<your_exisiting_aws_keypair_name> -c ddinfra:env=aws/agent-sandbox -c ddagent:apiKey=$DD_API_KEY -c ddinfra:aws/defaultPrivateKeyPath=$HOME/.ssh/id_rsa -s <your_name>-docker
```

## Quick start: Create an ECS EC2 (Windows/Linux) + Fargate (Linux) Cluster

```
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sso-agent-sandbox-account-admin -- pulumi up -c scenario=aws/ecs -c ddinfra:aws/defaultKeyPairName=<your_exisiting_aws_keypair_name> -c ddinfra:env=aws/agent-sandbox -c ddagent:apiKey=$DD_API_KEY -s <your_name>-ecs
```

## Quick start: Create an EKS (Linux/Windows) + Fargate (Linux) Cluster + Agent (Helm)

```
# You need to have a DD APIKey AND APPKey in variable DD_API_KEY / DD_APP_KEY
aws-vault exec sso-agent-sandbox-account-admin -- pulumi up -c scenario=aws/eks -c ddinfra:aws/defaultKeyPairName=<your_exisiting_aws_keypair_name> -c ddinfra:env=aws/agent-sandbox -c ddagent:apiKey=$DD_API_KEY -c ddagent:appKey=$DD_APP_KEY -s <your_name>-eks
```

## Troubleshooting

### Environment and configuration
The `setup.debug` invoke task will check for common mistakes such as key unavailable in configured AWS region, ssh-agent not running, invalid key format, and more.
```
aws-vault exec sso-agent-sandbox-account-admin -- inv setup.debug
aws-vault exec sso-agent-sandbox-account-admin -- inv setup --debug --no-interactive
```