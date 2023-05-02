from . import config
from .config import Config
import os
import invoke
import getpass
import subprocess
from invoke.context import Context
from typing import List, Optional, Dict, Any
import pathlib
from . import tool

default_public_path_key_name = "ddinfra:aws/defaultPublicKeyPath"


def deploy(
    _: Context,
    scenario_name: str,
    key_pair_required: bool = False,
    public_key_required: bool = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
    extra_flags: Dict[str, Any] = {},
):
    flags = extra_flags

    if install_agent is None:
        install_agent = tool.get_default_agent_install()
    flags["ddagent:deploy"] = install_agent

    cfg = config.get_config()
    flags[default_public_path_key_name] = _get_public_path_key_name(
        cfg, public_key_required
    )
    flags["scenario"] = scenario_name
    flags["ddagent:version"] = agent_version

    defaultKeyPairName = cfg.get_infra_aws().defaultKeyPairName
    flags["ddinfra:aws/defaultKeyPairName"] = defaultKeyPairName
    flags["ddinfra:env"] = "aws/sandbox"

    if install_agent:
        flags["ddagent:apiKey"] = _get_api_key()

    if key_pair_required and cfg.get_options().checkKeyPair:
        _check_key_pair(defaultKeyPairName)
    _deploy(stack_name, flags)


def _get_public_path_key_name(cfg: Config, require: bool) -> Optional[str]:
    defaultPublicKeyPath = cfg.get_infra_aws().defaultPublicKeyPath
    if require and defaultPublicKeyPath is None:
        raise invoke.Exit(
            f"Your scenario requires to define {default_public_path_key_name} in the configuration file"
        )
    return defaultPublicKeyPath


def _deploy(stack_name: Optional[str], flags: Dict[str, Any]) -> None:
    cmd_args = ["aws-vault", "exec", "sandbox-account-admin", "--", "pulumi", "up"]
    for key, value in flags.items():
        if value is not None and value != "":
            cmd_args.append("-c")
            cmd_args.append(f"{key}={value}")
    cmd_args.extend(["-s", _get_stack_name(stack_name, flags["scenario"])])
    cmd_args.extend(["-C", _get_root_path()])

    try:
        # use subprocess instead of context to allow interaction with pulumi up
        subprocess.check_call(cmd_args)
    except Exception as e:
        raise invoke.Exit(f"Error when running {cmd_args}: {e}")


def _get_root_path() -> str:
    folder = pathlib.Path(__file__).parent.resolve()
    return str(folder.parent)


def _get_api_key() -> str:
    api_key = os.getenv("E2E_API_KEY")
    if api_key is None or len(api_key) != 32:
        raise invoke.Exit(
            "Invalid API KEY. You must define the environment variable 'E2E_API_KEY'. "
            + "If you don't want an agent installation add '--no-install-agent'."
        )
    return api_key


def _check_key_pair(key_pair_to_search: Optional[str]):
    output = subprocess.check_output(["ssh-add", "-L"])
    key_pairs: List[str] = []
    output = output.decode("utf-8")
    for line in output.splitlines():
        parts = line.split(" ")
        if len(parts) > 0:
            key_pair_path = os.path.basename(parts[-1])
            key_pair = os.path.splitext(key_pair_path)[0]
            key_pairs.append(key_pair)

    if key_pair_to_search not in key_pairs:
        raise invoke.Exit(
            f"Your key pair value '{key_pair_to_search}' is not find in ssh-agent. "
            + f"You may have issue to connect to the remote instance. Possible values are \n{key_pairs}. "
            + f"You can skip this check by setting `checkKeyPair: false` in the config"
        )


def get_stack_name_prefix() -> str:
    return "invoke-"


def _get_stack_name(stack_name: Optional[str], scenario_name: str) -> str:
    if stack_name is None:
        scenario_name = scenario_name.replace("/", "-")
        stack_name = f"{scenario_name}-{getpass.getuser()}"
    return f"{get_stack_name_prefix()}{stack_name}"
