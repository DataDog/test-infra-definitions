import os
import os.path
from pathlib import Path
import getpass
import pyperclip

from invoke import task
from invoke.context import Context

from .config import Config, get_full_profile_path, get_local_config
from .tool import ask, info, warn

@task
def setup(ctx: Context):
    """
    Setup a local environment interactively
    """
    info("🤖 Install Pulumi")
    os.system("brew install pulumi/tap/pulumi")
    os.system("pulumi login --local")

    info("🤖 Let's configure your environment for e2e tests! Press ctrl+c to stop me")
    try:
        config = get_local_config()
    except:
        config = Config.parse_obj({})
        

    if config.configParams is None:
        config.configParams = Config.Params(
            aws=None,
            agent=None
        )
    
    # AWS config
    if config.configParams.aws is None:
        config.configParams.aws = Config.Params.Aws(
            keyPairName=None,
            publicKeyPath= None,
        )
        
    if config.configParams.aws.keyPairName is None: 
        config.configParams.aws.keyPairName = getpass.getuser()
    keyPairName = ask(f"🔑 Key pair name - stored in AWS Sandbox, default [{config.configParams.aws.keyPairName}]: ")
    if len(keyPairName) > 0:
        config.configParams.aws.keyPairName = keyPairName

    if config.options is None:
        config.options = Config.Options(
            checkKeyPair=False
        )
    default = "Y" if config.options.checkKeyPair else "N"
    checkKeyPair = ask(f"Did you create your SSH key on AWS and want me to check it is loaded on your ssh agent when creating manual environments or running e2e tests [Y/N]? Default [{default}]: ")
    if len(checkKeyPair) > 0:
        config.options.checkKeyPair = checkKeyPair.lower() == "y" or checkKeyPair.lower() == "yes"

    if config.configParams.aws.publicKeyPath is None:
        config.configParams.aws.publicKeyPath = str(Path.home().joinpath(".ssh", "id_ed25519.pub").absolute())
    default_public_key_path = config.configParams.aws.publicKeyPath
    while True:
        config.configParams.aws.publicKeyPath = default_public_key_path
        publicKeyPath = ask(f"🔑 Path to your public ssh key, default [{config.configParams.aws.publicKeyPath}]: ")
        if len(publicKeyPath) > 0:
            config.configParams.aws.publicKeyPath = publicKeyPath
        if os.path.isfile(config.configParams.aws.publicKeyPath):
            break
        warn(f"{config.configParams.aws.publicKeyPath} is not a valid ssh key")
    
    # Agent config
    if config.configParams.agent is None:
        config.configParams.agent = Config.Params.Agent(
            apiKey=None,
            appKey=None,
        )

    if config.configParams.agent.apiKey is None:
        config.configParams.agent.apiKey = '0' * 32
    default_api_key = config.configParams.agent.apiKey
    while True:
        config.configParams.agent.apiKey = default_api_key
        apiKey = ask(f"🐶 Datadog API key - default [{_get_safe_dd_key(config.configParams.agent.apiKey)}]: ")
        if len(apiKey) > 0:
            config.configParams.agent.apiKey = apiKey
        if len(config.configParams.agent.apiKey) == 32:
            break
        warn(f"Expecting API key of length 32, got {len(config.configParams.agent.apiKey)}")

    if config.configParams.agent.appKey is None:    
        config.configParams.agent.appKey = '0' * 40
    default_app_key = config.configParams.agent.appKey
    while True:
        config.configParams.agent.appKey = default_app_key

        app_Key = ask(f"🐶 Datadog APP key - default [{_get_safe_dd_key(config.configParams.agent.appKey)}]: ")
        if len(app_Key) > 0:
            config.configParams.agent.appKey = app_Key
        if len(config.configParams.agent.appKey) == 40:
            break
        warn(f"Expecting APP key of length 40, got {len(config.configParams.agent.appKey)}")

    config.save_to_local_config()
    cat_profile_command = f"cat {get_full_profile_path()}"
    pyperclip.copy(cat_profile_command)
    print(
        f"\nYou can run the following command to print your configuration: `{cat_profile_command}`. This command was copied to the clipboard\n"
    )
    
      
def _get_safe_dd_key(key: str) -> str: 
    if key == '0' * len(key):
        return key
    return '*' * len(key)   

    


