from . import config
import os
import invoke
import getpass
import subprocess
import shlex
from invoke.context import Context
from typing import Optional, Dict, Any


def deploy(
    ctx: Context,
    scenario_name: str,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
):
    flags = {}
    flags["scenario"] = scenario_name
    flags["ddagent:deploy"] = install_agent
    flags["ddagent:version"] = agent_version
    _deploy_with_config(ctx, flags)


def _deploy_with_config(ctx: Context, flags: Dict[str, Any]) -> None:
    cfg = config.get_config()
    flags["ddinfra:aws/defaultKeyPairName"] = cfg.key_pair
    flags["ddinfra:env"] = "aws/sandbox"

    if "ddagent:deploy" in flags and flags["ddagent:deploy"]:
        flags["ddagent:apiKey"] = _get_api_key()

    cmd_args = ["aws-vault", "exec", "sandbox-account-admin", "--", "pulumi", "up"]
    for key, value in flags.items():
        if value is not None and value != "":
            cmd_args.append("-c")
            cmd_args.append(shlex.quote(f"{key}={value}"))
    cmd_args.extend(["-s", _get_stack_name(flags["scenario"])])

    try:
        # use subprocess instead of context to allow interaction with pulumi up
        subprocess.check_call(cmd_args)
    except Exception as e:
        raise invoke.Exit(f"Error when running {cmd_args}: {e}")


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


def _get_stack_name(scenario_name: str) -> str:
    scenario_name = scenario_name.replace("/", "-")
    return f"{get_stack_name_prefix()}{scenario_name}-{getpass.getuser()}"
