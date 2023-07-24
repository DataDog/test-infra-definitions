import subprocess
from typing import List, Optional, Tuple

from invoke.context import Context
from invoke.exceptions import Exit
from pydantic import ValidationError

from . import config
from .tool import error, get_aws_wrapper, get_stack_name, get_stack_name_prefix, info


def destroy(
    ctx: Context,
    scenario_name: str,
    stack: Optional[str] = None,
    force_yes: Optional[bool] = False,
):
    """
    Destroy an environment
    """

    full_stack_name = get_stack_name(stack, scenario_name)
    short_stack_names, full_stack_names = _get_existing_stacks()
    force_destroy = "--yes --skip-preview" if force_yes else ""

    if len(short_stack_names) == 0:
        info("No stack to destroy")
        return

    try:
        cfg = config.get_local_config()
    except ValidationError as e:
        raise Exit(f"Error in config {config.get_full_profile_path()}:{e}")
    aws_account = cfg.get_aws().get_account()

    if stack is not None:
        if stack in short_stack_names:
            full_stack_name = f"{get_stack_name_prefix()}{stack}"
        else:
            error(f"Unknown stack '{stack}'")
            full_stack_name = None
    else:
        if full_stack_name not in full_stack_names:
            error(f"Unknown stack '{full_stack_name}'")
            full_stack_name = None

    if full_stack_name is None:
        error("Run this command with '--stack-name MY_STACK_NAME'. Available stacks are:")
        for stack_name in short_stack_names:
            error(f" {stack_name}")
    else:
        ctx.run(
            f"{get_aws_wrapper(aws_account)} -- pulumi destroy --remove -s {full_stack_name} {force_destroy}",
            pty=True,
        )


def _get_existing_stacks() -> Tuple[List[str], List[str]]:
    output = subprocess.check_output(["pulumi", "stack", "ls", "--all"])
    output = output.decode("utf-8")
    lines = output.splitlines()
    lines = lines[1:]  # skip headers
    stacks: List[str] = []
    full_stacks: List[str] = []
    stack_name_prefix = get_stack_name_prefix()
    for line in lines:
        # the stack has an asterisk if it is currently selected
        stack_name = line.split(" ")[0].rstrip("*")
        if stack_name.startswith(stack_name_prefix):
            full_stacks.append(stack_name)
            stack_name = stack_name[len(stack_name_prefix) :]
            stacks.append(stack_name)
    return stacks, full_stacks
