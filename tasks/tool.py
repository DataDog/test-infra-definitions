import getpass
import json
import platform
import subprocess
from termcolor import colored
from typing import Any, List, Optional

def ask(question: str) -> str:
    return input(colored(question, "blue"))

def debug(msg: str):
    print(colored(msg, "white"))

def info(msg: str):
    print(colored(msg, "green"))


def warn(msg: str):
    print(colored(msg, "yellow"))


def error(msg: str):
    print(colored(msg, "red"))


def get_os_families() -> List[str]:
    return [
        get_default_os_family(),
        "windows",
        "amazonlinux",
        "debian",
        "redhat",
        "suse",
        "fedora",
    ]


def get_default_os_family() -> str:
    return "ubuntu"


def get_default_agent_install() -> bool:
    return True


def get_repositories() -> List[str]:
    return [
        get_default_repository(),
        "staging",
        "testing",
        "trial",
    ]


def get_default_repository() -> str:
    return "prod"


def get_stack_name(stack_name: Optional[str], scenario_name: str) -> str:
    if stack_name is None:
        stack_name = scenario_name.replace("/", "-")
    # The scenario name cannot start with the stack name because ECS
    # stack name cannot start with 'ecs' or 'aws'
    return f"{get_stack_name_prefix()}{stack_name}"


def get_stack_name_prefix() -> str:
    user_name = f"{getpass.getuser()}-"
    return user_name.replace(".", "-") # EKS doesn't support '.'


def get_stack_json_outputs(full_stack_name: str) -> Any:
    output = subprocess.check_output(
        ["pulumi", "stack", "output", "--json", "-s", full_stack_name]
    )
    output = output.decode("utf-8")
    return json.loads(output)


def is_windows():
    return platform.system() == 'Windows'


class Connection:
    def __init__(self, stack_outputs: Any):
        connection: Any = stack_outputs["vm-connection"]
        self.host: str = connection["host"]
        self.user: str = connection["user"]
