import subprocess
from .tool import get_stack_name, get_stack_name_prefix, info, error
from typing import Optional, List, Tuple



def destroy(scenario_name: str, stack: Optional[str] = None):
    """
    Destroy an environment
    """

    full_stack_name = get_stack_name(stack, scenario_name)
    short_stack_names, full_stack_names = _get_existing_stacks()

    if len(short_stack_names) == 0:
        info("No stack to destroy")
        return

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
                full_stack_name,
            ]
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
        stack_name = line.split(" ")[0]
        if stack_name.startswith(stack_name_prefix):
            full_stacks.append(stack_name)
            stack_name = stack_name[len(stack_name_prefix):]
            stacks.append(stack_name)
    return stacks, full_stacks
