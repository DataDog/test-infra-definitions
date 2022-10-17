# Dynamic infrastructures for test

This repository contains IaC code based on Pulumi to provision dynamic test infrastructures for testing.

## Quick start guide

The first step is to download and install Pulumi CLI. Information can be found [here](https://www.pulumi.com/docs/get-started/install/).

For instance, on MacOS:
```
brew install pulumi/tap/pulumi
```

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

### Creating a stack

In this example, we're going to create an ECS Cluster:

```
# You need to have a DD APIKey in variable DD_API_KEY
aws-vault exec sandbox-account-admin -- pulumi up -c ddinfra:aws/defaultKeyPairName=<your_exisiting_aws_keypair_name> -c ddinfra:env=aws/sandbox -c ddagent:apiKey=$DD_API_KEY -C ./aws/ecs -s <your_name>-ecs-test
```