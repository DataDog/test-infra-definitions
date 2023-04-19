from termcolor import colored

def info(msg: str):
    print(colored(msg, "green"))

def warn(msg: str):
    print(colored(msg, "yellow"))

def error(msg: str):
    print(colored(msg, "red"))