import getpass
from termcolor import colored
from typing import List, Optional


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
    return False


def get_stack_name(stack_name: Optional[str], scenario_name: str) -> str:
    if stack_name is None:
        stack_name = scenario_name.replace("/", "-")
    return f"{stack_name}{get_stack_name_suffix()}"


def get_stack_name_suffix() -> str:
    return f"-{getpass.getuser()}"
