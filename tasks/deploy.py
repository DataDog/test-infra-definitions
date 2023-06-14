from pydantic import ValidationError
from . import config
from .config import Config, get_full_profile_path
import os
import subprocess
from invoke.context import Context
from invoke.exceptions import Exit 
from typing import Callable, List, Optional, Dict, Any
import pathlib
from . import tool

default_public_path_key_name = "ddinfra:aws/defaultPublicKeyPath"


def deploy(
    _: Context,
    scenario_name: str,
    key_pair_required: bool = False,
    public_key_required: bool = False,
    app_key_required: bool = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    extra_flags: Dict[str, Any] = {},
    use_fakeintake: Optional[bool] = False,
) -> str:
    flags = extra_flags

    if install_agent is None:
        install_agent = tool.get_default_agent_install()
    flags["ddagent:deploy"] = install_agent

    try:
        cfg = config.get_local_config()
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path()}:{e}")
    
    flags[default_public_path_key_name] = _get_public_path_key_name(
        cfg, public_key_required
    )
    flags["scenario"] = scenario_name
    flags["ddagent:version"] = agent_version
    flags["ddagent:fakeintake"] = use_fakeintake

    awsKeyPairName = cfg.get_aws().keyPairName
    flags["ddinfra:aws/defaultKeyPairName"] = awsKeyPairName
    flags["ddinfra:env"] = "aws/agent-sandbox"

    if install_agent:
        flags["ddagent:apiKey"] = _get_api_key(cfg)

    if key_pair_required and cfg.get_options().checkKeyPair:
        _check_key_pair(awsKeyPairName)

    # add stack params values
    stackParams = cfg.get_stack_params()
    for namespace in stackParams:
        for key, value in stackParams[namespace].items():
            flags[f"{namespace}:{key}"] = value


    if app_key_required:
        flags["ddagent:appKey"] = _get_app_key(cfg)

    return _deploy(stack_name, flags, debug)


def _get_public_path_key_name(cfg: Config, require: bool) -> Optional[str]:
    defaultPublicKeyPath = cfg.get_aws().publicKeyPath
    if require and defaultPublicKeyPath is None:
        raise Exit(
            f"Your scenario requires to define {default_public_path_key_name} in the configuration file"
        )
    return defaultPublicKeyPath


def _deploy(
    stack_name: Optional[str], flags: Dict[str, Any], debug: Optional[bool]
) -> str:
    cmd_args = [
        "aws-vault",
        "exec",
        "sso-agent-sandbox-account-admin",
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
        raise Exit(f"Error when running {cmd_args}: {e}")


def _get_root_path() -> str:
    folder = pathlib.Path(__file__).parent.resolve()
    return str(folder.parent)


def _get_api_key(cfg: Optional[Config]) -> str:
    return _get_key("API KEY", cfg, lambda c: c.get_agent().apiKey, "E2E_API_KEY", 32)


def _get_app_key(cfg: Optional[Config]) -> str:
    return _get_key("APP KEY", cfg, lambda c: c.get_agent().appKey, "E2E_APP_KEY", 40)


def _get_key(key_name: str, cfg: Optional[Config], get_key: Callable[[Config], Optional[str]], env_key_name: str, expected_size: int) -> str:
    key: Optional[str] = None

    # first try in config
    if cfg is not None:
        key = get_key(cfg)
    if key is None or len(key) == 0:
        # the try in env var
        key = os.getenv(env_key_name)
    if key is None or len(key) != expected_size:
        raise Exit(
            f"The scenario requires a valid {key_name} with a length of {expected_size} characters but none was found. You must define it in the config file"
        )
    return key


def _check_key_pair(key_pair_to_search: Optional[str]):
    if key_pair_to_search is None or key_pair_to_search == "":
        raise Exit(
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
        raise Exit(
            f"Your key pair value '{key_pair_to_search}' is not find in ssh-agent. "
            + f"You may have issue to connect to the remote instance. Possible values are \n{key_pairs}. "
            + "You can skip this check by setting `checkKeyPair: false` in the config"
        )
