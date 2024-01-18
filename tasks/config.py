from pathlib import Path
from typing import Dict, Optional

import yaml
from invoke.exceptions import Exit
from pydantic import BaseModel, Extra

from .tool import info

profile_filename = ".test_infra_config.yaml"


class Config(BaseModel, extra=Extra.forbid):
    class Params(BaseModel, extra=Extra.forbid):
        class Aws(BaseModel, extra=Extra.forbid):
            keyPairName: Optional[str]
            publicKeyPath: Optional[str]
            account: Optional[str]
            teamTag: Optional[str]

            def get_account(self) -> str:
                if self.account is None:
                    return "agent-sandbox"
                return self.account

        aws: Optional[Aws]

        class Agent(BaseModel, extra=Extra.forbid):
            apiKey: Optional[str]
            appKey: Optional[str]
            verifyCodeSignature: Optional[bool]

        agent: Optional[Agent]

        class Pulumi(BaseModel, extra=Extra.forbid):
            logLevel: Optional[int]
            logToStdErr: Optional[bool]

        pulumi: Optional[Pulumi]

        outputDir: Optional[str]

    configParams: Optional[Params] = None

    stackParams: Optional[Dict[str, Dict[str, str]]] = None

    class Options(BaseModel, extra=Extra.forbid):
        checkKeyPair: Optional[bool]

    options: Optional[Options] = None

    def get_options(self) -> Options:
        if self.options is None:
            return Config.Options(checkKeyPair=False)
        return self.options

    def get_aws(self) -> Params.Aws:
        default = Config.Params.Aws(keyPairName=None, publicKeyPath=None, account=None, teamTag=None)
        if self.configParams is None:
            return default
        if self.configParams.aws is None:
            return default
        return self.configParams.aws

    def get_agent(self) -> Params.Agent:
        default = Config.Params.Agent(apiKey=None, appKey=None)
        if self.configParams is None:
            return default
        if self.configParams.agent is None:
            return default
        return self.configParams.agent

    def get_stack_params(self) -> Dict[str, Dict[str, str]]:
        if self.stackParams is None:
            return {}
        return self.stackParams

    def save_to_local_config(self, config_path: Optional[str] = None):
        profile_path = get_full_profile_path(config_path)
        try:
            with open(profile_path, "w") as outfile:
                yaml.dump(self.dict(), outfile)
        except Exception as e:
            raise Exit(f"Error saving config file {profile_path}: {e}")
        info(f"Configuration file saved at {profile_path}")


def get_local_config(profile_path: Optional[str] = None) -> Config:
    profile_path = get_full_profile_path(profile_path)
    try:
        with open(profile_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            return Config.model_validate(config_dict)
    except FileNotFoundError:
        return Config.model_validate({})


def get_full_profile_path(profile_path: Optional[str] = None) -> str:
    if profile_path:
        return str(
            Path(profile_path).expanduser().absolute()
        )  # Return absolute path to config file, handle "~"" with expanduser
    return str(Path.home().joinpath(profile_filename))
