from . import config
from .config import Config
import os
import invoke
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
    debug: Optional[bool] = False,
    extra_flags: Dict[str, Any] = {},
) -> str:
    flags = extra_flags

    if install_agent is None:
        install_agent = tool.get_default_agent_install()
    flags["ddagent:deploy"] = install_agent

    cfg = config.get_local_config()
    flags[default_public_path_key_name] = _get_public_path_key_name(
        cfg, public_key_required
    )
    flags["scenario"] = scenario_name
    flags["ddagent:version"] = agent_version

    awsKeyPairName = cfg.get_aws().keyPairName
    flags["ddinfra:aws/defaultKeyPairName"] = awsKeyPairName
    flags["ddinfra:env"] = "aws/sandbox"

    if install_agent:
        flags["ddagent:apiKey"] = _get_api_key(cfg)

    if key_pair_required and cfg.get_options().checkKeyPair:
        _check_key_pair(awsKeyPairName)

    # add stack params values
    stackParams = cfg.get_stack_params()
    for namespace in stackParams:
        for key, value in stackParams[namespace].items():
            flags[f"{namespace}:{key}"] = value

    return _deploy(stack_name, flags, debug)


def _get_public_path_key_name(cfg: Config, require: bool) -> Optional[str]:
    defaultPublicKeyPath = cfg.get_aws().publicKeyPath
    if require and defaultPublicKeyPath is None:
        raise invoke.Exit(
            f"Your scenario requires to define {default_public_path_key_name} in the configuration file"
        )
    return defaultPublicKeyPath


def _deploy(
    stack_name: Optional[str], flags: Dict[str, Any], debug: Optional[bool]
) -> str:
    cmd_args = [
        "aws-vault",
        "exec",
        "sandbox-account-admin",
        "--",
        "pulumi",
        "up",
        "--yes",
    ]
    for key, value in flags.items():
        if value is not None and value != "":
            cmd_args.append("-c")
            cmd_args.append(f"{key}={value}")
    full_stack_name = tool.get_stack_name(stack_name, flags["scenario"])
    cmd_args.extend(["-s", full_stack_name])
    cmd_args.extend(["-C", _get_root_path()])

    if debug:
        cmd_args.extend(["-v", "3", "--debug"])

    try:
        # use subprocess instead of context to allow interaction with pulumi up
        subprocess.check_call(cmd_args)
        return full_stack_name
    except Exception as e:
        raise invoke.Exit(f"Error when running {cmd_args}: {e}")


def _get_root_path() -> str:
    folder = pathlib.Path(__file__).parent.resolve()
    return str(folder.parent)


def _get_api_key(cfg: Optional[Config]) -> str:
    # first try in config
    if cfg is not None and cfg.get_agent().apiKey is not None:
        return cfg.get_agent().apiKey
    # the try in env var
    api_key = os.getenv("E2E_API_KEY")
    if api_key is None or len(api_key) != 32:
        raise invoke.Exit(
            "Invalid API KEY. You must define the environment variable 'E2E_API_KEY'. "
            + "If you don't want an agent installation add '--no-install-agent'."
        )
    return api_key


def _check_key_pair(key_pair_to_search: Optional[str]):
    if key_pair_to_search is None or key_pair_to_search == "":
        raise invoke.Exit(
            "This scenario requires to define 'defaultKeyPairName' in the configuration file"
        )
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
