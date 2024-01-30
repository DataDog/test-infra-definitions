import getpass
import os
import os.path
from pathlib import Path
from typing import Optional

import pyperclip
from invoke.context import Context
from invoke.tasks import task

from . import doc
from .config import Config, get_full_profile_path, get_local_config
from .tool import ask, info, is_linux, is_windows, warn

available_aws_accounts = ["agent-sandbox", "sandbox", "agent-qa", "tse-playground"]


@task(help={"config_path": doc.config_path, "interactive": doc.interactive})
def setup(_: Context, config_path: Optional[str] = None, interactive: Optional[bool] = True) -> None:
    """
    Setup a local environment, interactively by default
    """
    pulumi_version, pulumi_up_to_date = _pulumi_version(ctx)
    if pulumi_up_to_date:
        info(f"Pulumi is up to date: {pulumi_version}")
    else:
        info("🤖 Install Pulumi")
        if is_windows():
            os.system("winget install pulumi")
        elif is_linux():
            os.system("curl -fsSL https://get.pulumi.com | sh")
        else:
            os.system("brew install pulumi/tap/pulumi")

    # install plugins
    os.system("pulumi --non-interactive plugin install")
    # login to local stack storage
    os.system("pulumi login --local")

    try:
        config = get_local_config(config_path)
    except Exception:
        config = Config.model_validate({})

    if interactive:
        info("🤖 Let's configure your environment for e2e tests! Press ctrl+c to stop me")
        # AWS config
        setupAWSConfig(config)
        # Agent config
        setupAgentConfig(config)

        config.save_to_local_config(config_path)

    if interactive:
        cat_profile_command = f"cat {get_full_profile_path(config_path)}"
        pyperclip.copy(cat_profile_command)
        print(
            f"\nYou can run the following command to print your configuration: `{cat_profile_command}`. This command was copied to the clipboard\n"
        )


def setupAWSConfig(config: Config):
    if config.configParams is None:
        config.configParams = Config.Params(aws=None, agent=None)
    if config.configParams.aws is None:
        config.configParams.aws = Config.Params.Aws(keyPairName=None, publicKeyPath=None, account=None, teamTag=None)

    # aws account
    if config.configParams.aws.account is None:
        config.configParams.aws.account = "agent-sandbox"
    default_aws_account = config.configParams.aws.account
    while True:
        config.configParams.aws.account = default_aws_account
        aws_account = ask(
            f"Which aws account do you want to create instances on? Default [{config.configParams.aws.account}], available [agent-sandbox|sandbox|tse-playground]: "
        )
        if len(aws_account) > 0:
            config.configParams.aws.account = aws_account
        if config.configParams.aws.account in available_aws_accounts:
            break
        warn(f"{config.configParams.aws.account} is not a valid aws account")

    # aws keypair name
    if config.configParams.aws.keyPairName is None:
        config.configParams.aws.keyPairName = getpass.getuser()
    keyPairName = ask(f"🔑 Key pair name - stored in AWS Sandbox, default [{config.configParams.aws.keyPairName}]: ")
    if len(keyPairName) > 0:
        config.configParams.aws.keyPairName = keyPairName

    # check keypair name
    if config.options is None:
        config.options = Config.Options(checkKeyPair=False)
    default_check_key_pair = "Y" if config.options.checkKeyPair else "N"
    checkKeyPair = ask(
        f"Did you create your SSH key on AWS and want me to check it is loaded on your ssh agent when creating manual environments or running e2e tests [Y/N]? Default [{default_check_key_pair}]: "
    )
    if len(checkKeyPair) > 0:
        config.options.checkKeyPair = checkKeyPair.lower() == "y" or checkKeyPair.lower() == "yes"

    # aws public key path
    if config.configParams.aws.publicKeyPath is None:
        config.configParams.aws.publicKeyPath = str(Path.home().joinpath(".ssh", "id_ed25519.pub").absolute())
    default_public_key_path = config.configParams.aws.publicKeyPath
    while True:
        config.configParams.aws.publicKeyPath = default_public_key_path
        publicKeyPath = ask(f"🔑 Path to your public ssh key, default [{config.configParams.aws.publicKeyPath}]: ")
        if len(publicKeyPath) > 0:
            config.configParams.aws.publicKeyPath = publicKeyPath
        if os.path.isfile(config.configParams.aws.publicKeyPath):
            break
        warn(f"{config.configParams.aws.publicKeyPath} is not a valid ssh key")

    # team tag
    if config.configParams.aws.teamTag is None:
        config.configParams.aws.teamTag = ""
    while True:
        msg = "🔖 What is your github team? This will tag all your resources by `team:<team>`. Use kebab-case format (example: agent-platform)"
        if len(config.configParams.aws.teamTag) > 0:
            msg += f". Default [{config.configParams.aws.teamTag}]"
        msg += ": "
        teamTag = ask(msg)
        if len(teamTag) > 0:
            config.configParams.aws.teamTag = teamTag
        if len(config.configParams.aws.teamTag) > 0:
            break
        warn("Provide a non-empty team")


def setupAgentConfig(config):
    if config.configParams.agent is None:
        config.configParams.agent = Config.Params.Agent(
            apiKey=None,
            appKey=None,
        )
    # API key
    if config.configParams.agent.apiKey is None:
        config.configParams.agent.apiKey = "0" * 32
    default_api_key = config.configParams.agent.apiKey
    while True:
        config.configParams.agent.apiKey = default_api_key
        apiKey = ask(f"🐶 Datadog API key - default [{_get_safe_dd_key(config.configParams.agent.apiKey)}]: ")
        if len(apiKey) > 0:
            config.configParams.agent.apiKey = apiKey
        if len(config.configParams.agent.apiKey) == 32:
            break
        warn(f"Expecting API key of length 32, got {len(config.configParams.agent.apiKey)}")
    # APP key
    if config.configParams.agent.appKey is None:
        config.configParams.agent.appKey = "0" * 40
    default_app_key = config.configParams.agent.appKey
    while True:
        config.configParams.agent.appKey = default_app_key

        app_Key = ask(f"🐶 Datadog APP key - default [{_get_safe_dd_key(config.configParams.agent.appKey)}]: ")
        if len(app_Key) > 0:
            config.configParams.agent.appKey = app_Key
        if len(config.configParams.agent.appKey) == 40:
            break
        warn(f"Expecting APP key of length 40, got {len(config.configParams.agent.appKey)}")


def _get_safe_dd_key(key: str) -> str:
    if key == "0" * len(key):
        return key
    return "*" * len(key)


def _pulumi_version(ctx: Context) -> (str, bool):
    """
    Returns True if pulumi is installed and up to date, False otherwise
    Will return True if PULUMI_SKIP_UPDATE_CHECK=1
    """
    try:
        out = ctx.run("pulumi version --logtostderr", hide=True)
    except UnexpectedExit as e:
        # likely pulumi command not found
        return "", False
    # The update message differs some between platforms so choose a common part
    up_to_date = "A new version of Pulumi is available" not in out.stderr
    return out.stdout.strip(), up_to_date