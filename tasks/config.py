import invoke
import subprocess
import os
import yaml
from pathlib import Path
from .tool import *
from typing import List, Dict, Any, Optional


class Config:
    key_pair: str
    defaultPublicKeyPath: Optional[str]


def get_config() -> Config:
    config_filename = ".test_infra_config.yaml"
    config_path = Path.home().joinpath(config_filename)
    try:
        with open(config_path) as f:
            content = f.read()
            config_dict = yaml.load(content, Loader=yaml.Loader)
            key_pair: str = _get_value(
                "ddinfra:aws/defaultKeyPairName", config_dict, config_path
            )
            check_key_pair: Optional[bool] = _get_optional_value(
                "checkKeyPair", config_dict
            )
            if check_key_pair is not None and check_key_pair:
                _check_key_pair(key_pair, config_path)

            defaultPublicKeyPath = _get_optional_value(
                "ddinfra:aws/defaultPublicKeyPath", config_dict
            )
            c = Config()
            c.key_pair = key_pair
            c.defaultPublicKeyPath = defaultPublicKeyPath
            return c
    except FileNotFoundError:
        raise invoke.Exit(f"Cannot find the configuration located at {config_path}")


def _get_value(key: str, config_dict: Dict[str, Any], config_path: Path) -> Any:
    if key not in config_dict:
        raise invoke.Exit(f"Cannot find the mandatory key {key} in {config_path}")
    return config_dict[key]


def _get_optional_value(key: str, config_dict: Dict[str, Any]) -> Optional[Any]:
    if key not in config_dict:
        return None
    return config_dict[key]


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
