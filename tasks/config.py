import invoke
import subprocess
import os
import json


def get_config():
    this_folder = os.path.dirname(os.path.realpath(__file__))
    config_path = os.path.join(this_folder, "config.json")
    if not os.path.exists(config_path):
        create_config(config_path)

    with open(config_path) as f:
        content = f.read()
        config = json.loads(content)
        check_ssh_key_exists(config["aws_key_pair"])
        return config


def create_config(config_path):
    print("Please enter the name of your AWS key pair value for example 'agent-ci-sandbox'.")
    key_pair = input()
    check_ssh_key_exists(key_pair)
    config = {
        "aws_key_pair": key_pair,
    }

    json_dump = json.dumps(config)
    with open(config_path, "w") as f:
        f.write(json_dump)
    print(f"The key pair value '{key_pair}' was saved into '{config_path}'")


def check_ssh_key_exists(key_pair):
    known_keys = get_known_ssh_key()
    if key_pair not in known_keys:
        raise invoke.Exit(
            f"The key '{key_pair}' is not listed by 'ssh-add -L'. Please run 'ssh-add PATH_TO_YOUR_KEY' and rerun this command.")


def get_known_ssh_key():
    output = subprocess.check_output(['ssh-add', '-L'])
    output = output.decode('utf-8')
    key_pairs = []
    for line in output.splitlines():
        parts = line.split(' ')
        if len(parts) > 0:
            key_pair_path = os.path.basename(parts[-1])
            key_pair = os.path.splitext(key_pair_path)[0]
            key_pairs.append(key_pair)
    return key_pairs
