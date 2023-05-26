import subprocess
from .tool import get_stack_name, get_stack_name_prefix, info, error
from typing import Optional, List



def destroy(scenario_name: str, stack: Optional[str] = None):
    """
    Destroy an environment
    """

    stack_name = get_stack_name(stack, scenario_name)

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

    if stack is None:
        error("Run this command with '--stack MY_STACK_NAME'. Available stacks are:")
        for stack_name in stacks:
            error(f" {stack_name}")
    else:
        stack = f"{get_stack_name_prefix()}{stack}"
        subprocess.call(
            [
                "aws-vault",
                "exec",
                "agent-sandbox-account-admin",
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
    stack_name_prefix = get_stack_name_prefix()
    for line in lines:
        stack_name = line.split(" ")[0]
        if stack_name.startswith(stack_name_prefix):
            stack_name = stack_name[len(stack_name_prefix):]
            stacks.append(stack_name)
    return stacks
