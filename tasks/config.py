import invoke
import subprocess
import os
import yaml
from pathlib import Path
from .tool import *
from typing import List

class Config:
    key_pair: str

def get_config() -> Config:
    config_filename = ".test_infra_config.yaml"
    config_path = Path.home().joinpath(config_filename)
    try:
        with open(config_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            key_name = "ddinfra:aws/defaultKeyPairName"
            if key_name not in config_dict:
               raise invoke.Exit(f"Cannot find the mandatory key {key_name} in {config_path}")
            key_pair = config_dict[key_name] 
            check_key_pair(key_pair)                    
            
            c = Config()
            c.key_pair = key_pair
            return c
    except FileNotFoundError:
        raise invoke.Exit(f"Cannot find the configuration located at {config_path}")
    

def check_key_pair(key_pair_to_search: str):
    output = ""
    try:
        output = subprocess.check_output(['ssh-add', '-L'])
    except:   
        # if ssh-add is not there don't display a warnings     
        return

    key_pairs: List[str] = []
    output = output.decode('utf-8')
    for line in output.splitlines():
        parts = line.split(' ')
        if len(parts) > 0:
            key_pair_path = os.path.basename(parts[-1])
            key_pair = os.path.splitext(key_pair_path)[0]
            key_pairs.append(key_pair)
            
    if key_pair_to_search not in key_pairs:
        warn(f"Your key pair value '{key_pair_to_search}' is not find in ssh-agent. " + 
                      f"You may have issue to connect to the remote instance. Possible values are \n{key_pairs}")
    
