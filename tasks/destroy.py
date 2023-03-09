from invoke import task
import subprocess
from .stack import get_scenario_folder_from_stack_name, is_known_stack


@task(help={'stack': "The name of the stack to destroy."})
def destroy(ctx, stack=""):
    """
    Destroy an environment
    """
    stacks = get_existing_stacks()

    if len(stacks) == 0:
        print("No stack to destroy")
        return

    # if there is a single stack delete it
    if stack == "" and len(stacks) == 1:
        stack = stacks[0]

    if stack != "" and stack not in stacks:
        print(f"Unknown stack '{stack}'")
        stack = ""

    if stack == "":
        print("Run this command with '--stack MY_STACK_NAME'. Available stacks are:")
        for stack_name in stacks:
            print(" - " + stack_name)
    else:
        scenario_folder = get_scenario_folder_from_stack_name(stack)

        # In order to destroy correctly the environment, `pulumi destroy` command has to be
        # run with the same configuration as when the environment was created. It is to make
        # sure the code path is the same if there is a condition on a value passed as
        # a command line parameters. That is why the code set the scenario folder.
        status = subprocess.call([
            "aws-vault", "exec", "sandbox-account-admin", "--",
            "pulumi", "destroy", "-C", scenario_folder, "-s", stack
        ])
        if status == 0:
            status = subprocess.call([
                "pulumi", "stack", "rm", "-s", stack, "-y"
            ])


def get_existing_stacks():
    output = subprocess.check_output(["pulumi", "stack", "ls", "--all"])
    output = output.decode('utf-8')
    lines = output.splitlines()
    lines = lines[1:]  # skip headers
    stacks = []
    for line in lines:
        stack_name = line.split(" ")[0]
        if is_known_stack(stack_name):
            stacks.append(stack_name)
    return stacks
