import invoke
import subprocess
import os
import yaml
from pathlib import Path
from .tool import *
from typing import List, Optional
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
        default = Config.Params.DDInfra.Aws(defaultKeyPairName=None, defaultPublicKeyPath=None)
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
        raise invoke.Exit(f"Cannot find the configuration located at {config_path}")
    except ValidationError as e:
        raise invoke.Exit(f"Error in config {config_path}:{e}")


def _check_key_pair(key_pair_to_search: str, config_path: Path):
    output = subprocess.check_output(["ssh-add", "-L"])
    key_pairs: List[str] = []
    output = output.decode("utf-8")
    for line in output.splitlines():
        parts = line.split(" ")
        if len(parts) > 0:
            key_pair_path = os.path.basename(parts[-1])
            key_pair = os.path.splitext(key_pair_path)[0]
            key_pairs.append(key_pair)

    if key_pair_to_search not in key_pairs:
        raise invoke.Exit(
            f"Your key pair value '{key_pair_to_search}' is not find in ssh-agent. "
            + f"You may have issue to connect to the remote instance. Possible values are \n{key_pairs}. "
            + f"You can skip this check by setting `checkKeyPair: false` in {config_path}"
        )
