import invoke
import yaml
from pathlib import Path
from .tool import *
from typing import Optional
from pydantic import BaseModel, Extra, ValidationError


class Config(BaseModel, extra=Extra.forbid):
    class Params(BaseModel, extra=Extra.forbid):
        class Aws(BaseModel, extra=Extra.forbid):
            keyPairName: Optional[str]
            publicKeyPath: Optional[str]

        aws: Optional[Aws]

        class Agent(BaseModel, extra=Extra.forbid):
            apiKey: Optional[str]
        
        agent: Optional[Agent]

    configParams: Optional[Params]

    stackParams: Optional[dict]

    class Options(BaseModel, extra=Extra.forbid):
        checkKeyPair: bool

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
            apiKey=None
        )
        if self.configParams == None:
            return default
        if self.configParams.agent == None:
            return default
        return self.configParams.agent


def get_local_config() -> Config:
    profile_filename = ".test_infra_config.yaml"
    profile_path = Path.home().joinpath(profile_filename)
    try:
        with open(profile_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            return Config.parse_obj(config_dict)

    except FileNotFoundError:
        return Config.parse_obj({})
    except ValidationError as e:
        raise invoke.Exit(f"Error in config {profile_path}:{e}")

# @task 