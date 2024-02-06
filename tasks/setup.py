import base64
import getpass
import json
import os
import os.path
from pathlib import Path
from typing import NamedTuple, Optional, Tuple

import pyperclip
from invoke.context import Context
from invoke.exceptions import Exit, UnexpectedExit
from invoke.tasks import task

from . import doc
from .config import Config, get_full_profile_path, get_local_config
from .tool import ask, debug, error, info, is_linux, is_windows, warn

available_aws_accounts = ["agent-sandbox", "sandbox", "agent-qa", "tse-playground"]


@task(help={"config_path": doc.config_path, "interactive": doc.interactive, "debug": doc.debug}, default=True)
def setup(
    ctx: Context, config_path: Optional[str] = None, interactive: Optional[bool] = True, debug: Optional[bool] = False
) -> None:
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

    if debug:
        debug_env(ctx, config_path=config_path)

    if interactive:
        cat_profile_command = f"cat {get_full_profile_path(config_path)}"
        pyperclip.copy(cat_profile_command)
        print(
            f"\nYou can run the following command to print your configuration: `{cat_profile_command}`. This command was copied to the clipboard\n"
        )


def setupAWSConfig(config: Config):
    if config.configParams is None:
        config.configParams = Config.Params(aws=None, agent=None, pulumi=None)
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


def _pulumi_version(ctx: Context) -> Tuple[str, bool]:
    """
    Returns True if pulumi is installed and up to date, False otherwise
    Will return True if PULUMI_SKIP_UPDATE_CHECK=1
    """
    try:
        out = ctx.run("pulumi version --logtostderr", hide=True)
    except UnexpectedExit:
        # likely pulumi command not found
        return "", False
    if out is None:
        return "", False
    # The update message differs some between platforms so choose a common part
    up_to_date = "A new version of Pulumi is available" not in out.stderr
    return out.stdout.strip(), up_to_date


def ssh_fingerprint_to_bytes(fingerprint: str) -> bytes:
    # EXAMPLE: 256 SHA1:41jsg4Z9lgylj6/zmhGxtZ6/qZs testname (ED25519)
    out = fingerprint.strip().split(' ')[1].split(':')[1]
    # ssh leaves out padding but python will ignore extra padding so add the missing padding
    return base64.b64decode(out + '==')


KeyFingerprint = NamedTuple('KeyFingerprint', [('md5', str), ('sha1', str), ('sha256', str)])


class KeyInfo(NamedTuple('KeyFingerprint', [('path', str), ('fingerprint', KeyFingerprint)])):
    def in_ssh_agent(self, ctx):
        out = ctx.run("ssh-add -l", hide=True)
        out = ssh_fingerprint_to_bytes(out.stdout.strip())
        return self.match(out)

    def match(self, fingerprint: bytes):
        for f in self.fingerprint:
            if f == fingerprint:
                return True
        return False

    def match_ec2_keypair(self, keypair):
        # EC2 uses a different fingerprint hash/format depending on the key type and the key's origin
        # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-keys.html
        ec2_fingerprint = keypair["KeyFingerprint"]
        if ':' in ec2_fingerprint:
            ec2_fingerprint = bytes.fromhex(ec2_fingerprint.replace(':', ''))
        else:
            ec2_fingerprint = base64.b64decode(ec2_fingerprint + '==')
        return self.match(ec2_fingerprint)

    @classmethod
    def from_path(cls, ctx, path):
        # Make sure the key is ascii
        with open(path, 'rb') as f:
            firstline = f.readline()
            if b'\0' in firstline:
                raise ValueError(f"Key file {path} is not ascii, it may be in utf-16, please convert it to ascii")
            # EC2 uses a different fingerprint hash/format depending on the key type and the key's origin
            # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/verify-keys.html
            if b'SSH' in firstline or firstline.startswith(b'ssh-'):

                def getfingerprint(fmt, path):
                    out = ctx.run(f"ssh-keygen -l -E {fmt} -f \"{path}\"", hide=True)
                    return ssh_fingerprint_to_bytes(out.stdout.strip())

            elif b'BEGIN' in firstline:

                def getfingerprint(fmt, path):
                    out = ctx.run(
                        f'openssl pkcs8 -in "{path}" -inform PEM -outform DER -topk8 -nocrypt | openssl {fmt} -c',
                        hide=True,
                    )
                    # EXAMPLE: (stdin)= e3:a8:bc:0a:3a:54:9f:b8:be:6e:75:8c:98:26:8e:3d:8e:e9:d0:69
                    out = out.stdout.strip().split(' ')[1]
                    return bytes.fromhex(out.replace(':', ''))

            else:
                raise ValueError(f"Key file {path} is not a valid ssh key")
        # aws returns fingerprints in different formats so get a couple
        fingerprints = dict()
        for fmt in KeyFingerprint._fields:
            fingerprints[fmt] = getfingerprint(fmt, path)
        return cls(path=path, fingerprint=KeyFingerprint(**fingerprints))


def load_ec2_keypairs(ctx: Context) -> dict:
    out = ctx.run("aws ec2 describe-key-pairs --output json", hide=True)
    if not out or out.exited != 0:
        warn("No AWS keypair found, please create one")
        return {}
    jso = json.loads(out.stdout)
    keypairs = jso.get("KeyPairs", None)
    if keypairs is None:
        warn("No AWS keypair found, please create one")
        return {}
    return keypairs


def find_matching_ec2_keypair(ctx: Context, keypairs: dict, path: Path) -> Tuple[Optional[KeyInfo], Optional[dict]]:
    if not os.path.exists(path):
        warn(f"WARNING: Key file {path} does not exist")
        return None, None
    info = KeyInfo.from_path(ctx, path)
    for keypair in keypairs:
        if info.match_ec2_keypair(keypair):
            return info, keypair
    return None, None


def get_ssh_keys():
    root = Path.home().joinpath(".ssh")
    return list(map(root.joinpath, os.listdir(root)))


def _check_key(ctx: Context, keyinfo: KeyInfo, keypair: dict, configuredKeyPairName: str):
    if keypair["KeyName"] != configuredKeyPairName:
        warn("WARNING: Key name does not match configured keypair name. This key will not be used for provisioning.")
    if not keyinfo.in_ssh_agent(ctx):
        warn("WARNING: Key missing from ssh-agent. This key will not be used for connections.")
    if "rsa" not in keypair["KeyType"].lower():
        warn("WARNING: Key type is not RSA. This key cannot be used to decrypt Windows RDP credentials.")


@task(help={"config_path": doc.config_path})
def debug_keys(ctx: Context, config_path: Optional[str] = None):
    """
    Debug E2E and test-infra-definitions SSH keys
    """
    # Ensure ssh-agent is running
    try:
        ctx.run("ssh-add -l", hide=True)
    except UnexpectedExit as e:
        error(f"{e}")
        error("ssh-agent not available or no keys are loaded, please start it and load your keys")
        raise Exit(code=1)

    found = False
    keypairs = load_ec2_keypairs(ctx)

    info("Checking for valid SSH key configuration")

    # Get keypair name
    try:
        config = get_local_config(config_path)
    except Exception as e:
        error(f"{e}")
        error("Failed to load config")
        raise Exit(code=1)
    if config.configParams is None:
        error("configParams missing from config")
        raise Exit(code=1)
    if config.configParams.aws is None:
        error("configParams.aws missing from config")
        raise Exit(code=1)
    awsConf = config.configParams.aws
    keypair_name = awsConf.keyPairName or ""

    # lookup configured keypair
    info("Checking configured keypair:")
    debug(f"\taws.keyPairName: {keypair_name}")
    debug(f"\taws.privateKeyPath: {awsConf.privateKeyPath}")
    debug(f"\taws.publicKeyPath: {awsConf.publicKeyPath}")
    for keypair in keypairs:
        if keypair["KeyName"] == keypair_name:
            info("Configured keyPairName found in aws!")
            debug(json.dumps(keypair, indent=4))
            break
    else:
        warn("WARNING: Configured keyPairName missing from aws!")
    for keyname in ["privateKeyPath", "publicKeyPath"]:
        keypair_path = getattr(awsConf, keyname)
        if keypair_path is None:
            continue
        keyinfo, keypair = find_matching_ec2_keypair(ctx, keypairs, keypair_path)
        if keyinfo is not None and keypair is not None:
            info(f"Configured {keyname} found in aws!")
            debug(json.dumps(keypair, indent=4))
            _check_key(ctx, keyinfo, keypair, keypair_name)
            found = True
        else:
            warn(f"WARNING: Configured {keyname} missing from aws!")

    print()

    info("Checking if any SSH key is configured in aws")

    # check all keypairs
    for keypath in get_ssh_keys():
        try:
            keyinfo, keypair = find_matching_ec2_keypair(ctx, keypairs, keypath)
        except (ValueError, UnexpectedExit) as e:
            if 'not a valid ssh key' in str(e):
                continue
            warn(f'WARNING: {e}')
            continue
        if keyinfo is not None and keypair is not None:
            info(f"Found '{keypair['KeyName']}' matches: {keypath}")
            debug(json.dumps(keypair, indent=4))
            _check_key(ctx, keyinfo, keypair, keypair_name)
            print()
            found = True

    if not found:
        error("No matching keypair found in aws!")
        info(
            "If this is unexpected, confirm that your aws credential's region matches the region you uploaded your key to."
        )
        raise Exit(code=1)


@task(name="debug", help={"config_path": doc.config_path})
def debug_env(ctx, config_path: Optional[str] = None):
    """
    Debug E2E and test-infra-definitions required tools and configuration
    """
    # check pulumi found
    try:
        out = ctx.run("pulumi version", hide=True)
    except UnexpectedExit as e:
        error(f"{e}")
        error("Pulumi CLI not found, please install it: https://www.pulumi.com/docs/get-started/install/")
        raise Exit(code=1)
    info(f"Pulumi version: {out.stdout.strip()}")

    # check awscli version
    out = ctx.run("aws --version", hide=True)
    if not out.stdout.startswith("aws-cli/2"):
        error(f"Detected invalid awscli version: {out.stdout}")
        info(
            "Please remove the current version and install awscli v2: https://docs.aws.amazon.com/cli/latest/userguide/cliv2-migration-instructions.html"
        )
        raise Exit(code=1)
    info(f"AWS CLI version: {out.stdout.strip()}")

    # check aws-vault found
    try:
        out = ctx.run("aws-vault --version", hide=True)
    except UnexpectedExit as e:
        error(f"{e}")
        error("aws-vault not found, please install it")
        raise Exit(code=1)
    info(f"aws-vault version: {out.stderr.strip()}")

    print()

    # Check if aws creds are valid
    try:
        out = ctx.run("aws sts get-caller-identity", hide=True)
    except UnexpectedExit as e:
        error(f"{e}")
        error("No AWS credentials found or they are expired, please configure and/or login")
        raise Exit(code=1)

    # Show AWS account info
    info("Logged-in aws account info:")
    for env in ["AWS_VAULT", "AWS_REGION"]:
        val = os.environ.get(env, None)
        if val is None:
            raise Exit(f"Missing env var {env}, please login with awscli/aws-vault", 1)
        info(f"\t{env}={val}")

    print()

    # Check aws-vault profile name, some invoke taskes hard code this value.
    expected_profile = 'sso-agent-sandbox-account-admin'
    out = ctx.run("aws-vault list", hide=True)
    if expected_profile not in out.stdout:
        warn(f"WARNING: expected profile {expected_profile} missing from aws-vault. Some invoke tasks may fail.")
        print()

    debug_keys(ctx, config_path=config_path)
