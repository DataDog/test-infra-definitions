import invoke
import yaml
from pathlib import Path
from .tool import *
from typing import Optional
from pydantic import BaseModel, Extra, ValidationError


class Config(BaseModel, extra=Extra.forbid):
    class Params(BaseModel, extra=Extra.forbid):
        class DDInfra(BaseModel, extra=Extra.forbid):
            class Aws(BaseModel, extra=Extra.forbid):
                defaultKeyPairName: Optional[str]
                defaultPublicKeyPath: Optional[str]

            aws: Optional[Aws]

        ddinfra: Optional[DDInfra]

    stackParams: Optional[Params]

    class Options(BaseModel, extra=Extra.forbid):
        checkKeyPair: bool

    options: Optional[Options]

    def get_options(self) -> Options:
        if self.options is None:
            return Config.Options(checkKeyPair=False)
        return self.options

    def get_infra_aws(self) -> Params.DDInfra.Aws:
        default = Config.Params.DDInfra.Aws(
            defaultKeyPairName=None, defaultPublicKeyPath=None
        )
        if self.stackParams == None:
            return default
        if self.stackParams.ddinfra == None:
            return default
        if self.stackParams.ddinfra.aws is None:
            return default
        return self.stackParams.ddinfra.aws


def get_config() -> Config:
    config_filename = ".test_infra_config.yaml"
    config_path = Path.home().joinpath(config_filename)
    try:
        with open(config_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            return Config.parse_obj(config_dict)

    except FileNotFoundError:
        return Config.parse_obj({})
    except ValidationError as e:
        raise invoke.Exit(f"Error in config {config_path}:{e}")
