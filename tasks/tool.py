from termcolor import colored

def info(msg):
    print(colored(msg, "green"))

def warn(msg):
    print(colored(msg, "yellow"))

def error(msg):
    print(colored(msg, "red"))