import getpass
import invoke

stack_vm = "vm"
stacks = {stack_vm: ("./aws/scenarios/vm/", "aws-vm")}


def get_stack_name(scenario_name):
    stack_name = stacks[scenario_name][1]
    return getpass.getuser() + "-" + stack_name


def get_scenario_folder(scenario_name):
    return stacks[scenario_name][0]


def get_scenario_folder_from_stack_name(full_stack_name):
    scenario_folder = find_scenario_folder(full_stack_name)
    if scenario_folder is not None:
        return scenario_folder
    raise invoke.Exit(
        f"Cannot find the associated scenario for the stack '{full_stack_name}'. Was this stack created outside invoke tasks?")


def find_scenario_folder(full_stack_name):
    for scenario_folder, stack_name in stacks.values():
        if full_stack_name.endswith(stack_name):
            return scenario_folder
    return None


def is_known_stack(full_stack_name):
    return find_scenario_folder(full_stack_name) is not None
