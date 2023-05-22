import invoke
import yaml
from pathlib import Path
from .tool import *
from typing import Dict, Optional
from pydantic import BaseModel, Extra, ValidationError


profile_filename = ".test_infra_config.yaml"

class Config(BaseModel, extra=Extra.forbid):
    class Params(BaseModel, extra=Extra.forbid):
        class Aws(BaseModel, extra=Extra.forbid):
            keyPairName: Optional[str]
            publicKeyPath: Optional[str]

        aws: Optional[Aws]

        class Agent(BaseModel, extra=Extra.forbid):
            apiKey: Optional[str]
            appKey: Optional[str]
        
        agent: Optional[Agent]

    configParams: Optional[Params]

    stackParams: Optional[Dict[str, Dict[str,str]]]

    class Options(BaseModel, extra=Extra.forbid):
        checkKeyPair: Optional[bool]

    options: Optional[Options]

    def get_options(self) -> Options:
        if self.options is None:
            return Config.Options(checkKeyPair=False)
        return self.options

    def get_aws(self) -> Params.Aws:
        default = Config.Params.Aws(
            keyPairName=None, publicKeyPath=None
        )
        if self.configParams == None:
            return default
        if self.configParams.aws is None:
            return default
        return self.configParams.aws
  
    def get_agent(self) -> Params.Agent:
        default = Config.Params.Agent(
            apiKey=None,
            appKey=None
        )
        if self.configParams == None:
            return default
        if self.configParams.agent == None:
            return default
        return self.configParams.agent
    
    def get_stack_params(self) -> Dict[str, Dict[str,str]]:
        if self.stackParams == None:
            return {}
        return self.stackParams

    def save_to_local_config(self):
        profile_path = get_full_profile_path()
        try:
            with open(profile_path, 'w') as outfile:
                yaml.dump(self.dict(), outfile)
        except e:
            raise invoke.Exit(f"Error saving config file {profile_path}: {e}")
        info(f"Configuration file saved at {profile_path}")

def get_local_config() -> Config:
    profile_path = get_full_profile_path()
    try:
        with open(profile_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            return Config.parse_obj(config_dict)
    except FileNotFoundError:
        return Config.parse_obj({})
    except ValidationError as e:
        raise invoke.Exit(f"Error in config {profile_path}:{e}")
    
def get_full_profile_path() -> str:
    return Path.home().joinpath(profile_filename)