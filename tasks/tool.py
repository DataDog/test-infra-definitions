from termcolor import colored
from typing import List


def info(msg: str):
    print(colored(msg, "green"))


def warn(msg: str):
    print(colored(msg, "yellow"))


def error(msg: str):
    print(colored(msg, "red"))


def get_os_families() -> List[str]:
    return [
        get_default_os_family(),
        "Windows",
        "AmazonLinux",
        "Debian",
        "RedHat",
        "Suse",
        "Fedora",
    ]


def get_default_os_family() -> str:
    return "Ubuntu"


def get_default_agent_install() -> bool:
    return False
