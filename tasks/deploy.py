import os
import subprocess
from typing import Any, Callable, Dict, List, Optional

from invoke.context import Context
from invoke.exceptions import Exit
from pydantic import ValidationError

from . import config, tool
from .config import Config, get_full_profile_path

default_public_path_key_name = "ddinfra:aws/defaultPublicKeyPath"


def deploy(
    ctx: Context,
    scenario_name: str,
    key_pair_required: bool = False,
    public_key_required: bool = False,
    app_key_required: bool = False,
    stack_name: Optional[str] = None,
    pipeline_id: Optional[str] = None,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    extra_flags: Optional[Dict[str, Any]] = None,
    use_fakeintake: Optional[bool] = False,
) -> str:
    flags = extra_flags
    if flags is None:
        flags = {}

    if install_agent is None:
        install_agent = tool.get_default_agent_install()
    flags["ddagent:deploy"] = install_agent

    try:
        cfg = config.get_local_config()
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path()}:{e}")

    flags[default_public_path_key_name] = _get_public_path_key_name(cfg, public_key_required)
    flags["scenario"] = scenario_name
    flags["ddagent:pipeline_id"] = pipeline_id
    flags["ddagent:version"] = agent_version
    flags["ddagent:fakeintake"] = use_fakeintake

    awsKeyPairName = cfg.get_aws().keyPairName
    flags["ddinfra:aws/defaultKeyPairName"] = awsKeyPairName
    aws_account = cfg.get_aws().get_account()
    flags["ddinfra:env"] = "aws/" + aws_account

    if cfg.get_aws().teamTag is None or cfg.get_aws().teamTag == "":
        raise Exit(
            "Error in config, missing configParams.aws.teamTag. Run `inv setup` again and provide a valid team name"
        )
    flags["ddinfra:extraResourcesTags"] = f"team:{cfg.get_aws().teamTag}"

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

    return _deploy(ctx, stack_name, flags, debug)


def _get_public_path_key_name(cfg: Config, require: bool) -> Optional[str]:
    defaultPublicKeyPath = cfg.get_aws().publicKeyPath
    if require and defaultPublicKeyPath is None:
        raise Exit(f"Your scenario requires to define {default_public_path_key_name} in the configuration file")
    return defaultPublicKeyPath


# creates a stack with the given stack_name if it doesn't already exists
def _create_stack(ctx: Context, stack_name: str, global_flags: str):
    result = ctx.run(f"pulumi {global_flags} stack ls --all", hide="stdout")
    if not result:
        return

    stacks = result.stdout.splitlines()[1:]  # skip header
    for stack in stacks:
        # the stack has an asterisk if it is currently selected
        ls_stack_name = stack.split(" ")[0].rstrip("*")
        if ls_stack_name == stack_name:
            return

    ctx.run(f"pulumi {global_flags} stack init --no-select {stack_name}")


def _deploy(
    ctx: Context,
    stack_name: Optional[str],
    flags: Dict[str, Any],
    debug: Optional[bool],
) -> str:
    stack_name = tool.get_stack_name(stack_name, flags["scenario"])
    aws_account = flags["ddinfra:env"][len("aws/") :]
    global_flags = ""
    up_flags = ""

    # Checking root path
    root_path = tool._get_root_path()
    if root_path != os.getcwd():
        global_flags += f" -C {root_path}"

    # Building run func parameters
    for key, value in flags.items():
        if value is not None and value != "":
            up_flags += f" -c {key}={value}"

    if debug:
        global_flags += " --logflow --logtostderr -v 3"
        up_flags += " --debug"

    _create_stack(ctx, stack_name, global_flags)

    cmd = f"{tool.get_aws_wrapper(aws_account)} -- pulumi {global_flags} up --yes -s {stack_name} {up_flags}"
    ctx.run(cmd, pty=True)
    return stack_name


def _get_api_key(cfg: Optional[Config]) -> str:
    return _get_key("API KEY", cfg, lambda c: c.get_agent().apiKey, "E2E_API_KEY", 32)


def _get_app_key(cfg: Optional[Config]) -> str:
    return _get_key("APP KEY", cfg, lambda c: c.get_agent().appKey, "E2E_APP_KEY", 40)


def _get_key(
    key_name: str,
    cfg: Optional[Config],
    get_key: Callable[[Config], Optional[str]],
    env_key_name: str,
    expected_size: int,
) -> str:
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
        raise Exit("This scenario requires to define 'defaultKeyPairName' in the configuration file")
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
