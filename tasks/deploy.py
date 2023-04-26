from . import config
from .config import Config
import os
import invoke
import getpass
import subprocess
from invoke.context import Context
from typing import Optional, Dict, Any
import pathlib
from . import tool

default_public_path_key_name = "ddinfra:aws/defaultPublicKeyPath"


def deploy(
    _: Context,
    scenario_name: str,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
    os_family: Optional[str] = None,
):
    flags = {}

    if install_agent is None:
        install_agent = tool.get_default_agent_install()
    flags["ddagent:deploy"] = install_agent

    os_family = _get_os_family(os_family)
    flags["ddinfra:osFamily"] = os_family

    cfg = config.get_config()
    flags[default_public_path_key_name] = _default_public_path_key_name(cfg, os_family)
    flags["scenario"] = scenario_name
    flags["ddagent:version"] = agent_version

    flags["ddinfra:aws/defaultKeyPairName"] = cfg.get_infra_aws().defaultKeyPairName
    flags["ddinfra:env"] = "aws/sandbox"

    if install_agent:
        flags["ddagent:apiKey"] = _get_api_key()

    _deploy(stack_name, flags)


def _get_os_family(os_type: Optional[str]) -> str:
    os_types = tool.get_os_families()
    if os_type is None:
        os_type = tool.get_default_os_family()
    if os_type not in os_types:
        raise invoke.Exit(
            f"the os type '{os_type}' is not supported. Possibles values are {os_types}"
        )
    return os_type


def _default_public_path_key_name(cfg: Config, os_family: str) -> Optional[str]:
    defaultPublicKeyPath = cfg.get_infra_aws().defaultPublicKeyPath
    if os_family == "Windows" and defaultPublicKeyPath is None:
        raise invoke.Exit(
            f"You must set {default_public_path_key_name} when using this operating system"
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
    api_key = os.getenv("DD_API_KEY")
    if api_key is None or len(api_key) != 32:
        raise invoke.Exit(
            "Invalid API KEY. You must define the environment variable 'DD_API_KEY'. "
            + "If you don't want an agent installation add '--no-install-agent'."
        )
    return api_key


def get_stack_name_prefix() -> str:
    return "invoke-"


def _get_stack_name(stack_name: Optional[str], scenario_name: str) -> str:
    if stack_name is None:
        scenario_name = scenario_name.replace("/", "-")
        stack_name = f"{scenario_name}-{getpass.getuser()}"
    return f"{get_stack_name_prefix()}{stack_name}"
