import invoke
import subprocess
import os
import json
from pathlib import Path
from .tool import *

def get_config():
    config_filename = ".test-infra"
    config_path = Path.home().joinpath(config_filename)
    try:
        with open(config_path) as f:
            content = f.read()
            config = json.loads(content)
            key = "aws_key_pair"
            if key not in config:
               raise invoke.Exit(f"Cannot find the mandatory key {key} in {config_path}")
            else:
                key_pair = config[key] 
                check_key_pair(key_pair)                    
            
            return config
    except FileNotFoundError:
        raise invoke.Exit(f"Cannot find the configuration located at {config_path}")
    

def check_key_pair(key_pair_to_search):
    output = ""
    try:
        output = subprocess.check_output(['ssh-add', '-L'])
    except:   
        # if ssh-add is not there don't display a warnings     
        return

    key_pairs = []
    output = output.decode('utf-8')
    for line in output.splitlines():
        parts = line.split(' ')
        if len(parts) > 0:
            key_pair_path = os.path.basename(parts[-1])
            key_pair = os.path.splitext(key_pair_path)[0]
            key_pairs.append(key_pair)
            
    if key_pair_to_search not in key_pairs:
        warn(f"Your key pair value '{key_pair}' is not find in ssh-agent. " + 
                      f"You may have issue to connect to the remote instance. Possible values are \n{key_pairs}")
    
