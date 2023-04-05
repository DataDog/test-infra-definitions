from invoke import task
import subprocess
from . import deploy
from .tool import *

@task(help={'stack': "The name of the stack to destroy."})
def destroy(ctx, stack=None):
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
        status = subprocess.call([
            "aws-vault", "exec", "sandbox-account-admin", "--",
            "pulumi", "destroy", "-s", stack
        ])
        if status == 0:
            status = subprocess.call([
                "pulumi", "stack", "rm", "-s", stack, "-y"
            ])


def _get_existing_stacks():
    output = subprocess.check_output(["pulumi", "stack", "ls", "--all"])
    output = output.decode('utf-8')
    lines = output.splitlines()
    lines = lines[1:]  # skip headers
    stacks = []
    for line in lines:
        stack_name = line.split(" ")[0]
        if stack_name.startswith(deploy.stack_name_prefix):
            stacks.append(stack_name)
    return stacks
