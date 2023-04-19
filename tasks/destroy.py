from invoke import task
import subprocess
from .deploy import get_stack_name_prefix
from .tool import *
from invoke.context import Context
from typing import Optional, List


@task(help={"stack": "The name of the stack to destroy."})
def destroy(ctx: Context, stack: Optional[str] = None):
    """
    Destroy an environment
    """

    stacks = _get_existing_stacks()

    if len(stacks) == 0:
        info("No stack to destroy")
        return

    # if there is a single stack delete it
    if stack is None and len(stacks) == 1:
        stack = stacks[0]

    if stack is not None and stack not in stacks:
        error(f"Unknown stack '{stack}'")
        stack = None

    if stack == None:
        error("Run this command with '--stack MY_STACK_NAME'. Available stacks are:")
        for stack_name in stacks:
            error(" - " + stack_name)
    else:
        subprocess.call(
            [
                "aws-vault",
                "exec",
                "sandbox-account-admin",
                "--",
                "pulumi",
                "destroy",
                "--remove",
                "-s",
                stack,
            ]
        )


def _get_existing_stacks() -> List[str]:
    output = subprocess.check_output(["pulumi", "stack", "ls", "--all"])
    output = output.decode("utf-8")
    lines = output.splitlines()
    lines = lines[1:]  # skip headers
    stacks: List[str] = []
    for line in lines:
        stack_name = line.split(" ")[0]
        if stack_name.startswith(get_stack_name_prefix()):
            stacks.append(stack_name)
    return stacks
