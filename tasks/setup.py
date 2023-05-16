import os.path
from pathlib import Path
import getpass

from invoke import task
from invoke.context import Context

from .config import Config, profile_filename
from .tool import ask, debug, info, warn

@task
def setup(ctx: Context):
    """
    Setup a local environment interactively
    """
    info("🤖 Let's configure your environment for e2e tests! Press ctrl+c to stop me")
    config = Config()
    config.configParams = Config.Params()
    config.configParams.agent = Config.Params.Agent()
    config.configParams.aws = Config.Params.Aws()

    config.configParams.aws.keyPairName = getpass.getuser()
    keyPairName = ask(f"🔑 Key pair name - stored in AWS Sandbox, default [{config.configParams.aws.keyPairName}]: ")
    if len(keyPairName) > 0:
        config.configParams.aws.keyPairName = keyPairName

    checkKeyPair = ask("Did you create your SSH key on AWS and want me to check it is loaded on your ssh agent [Y/N]? Default No: ")
    if len(checkKeyPair) > 0:
        config.options = Config.Options()
        config.options.checkKeyPair = checkKeyPair.lower() == "y" or checkKeyPair.lower() == "yes"

    config.configParams.aws.publicKeyPath = Path.home().joinpath(".ssh", "id_ed25519.pub")
    publicKeyPath = ask(f"🔑 Path to your public ssh key: default [{config.configParams.aws.publicKeyPath}]")
    if len(publicKeyPath) > 0:
        config.configParams.aws.publicKeyPath = publicKeyPath
    while not os.path.isfile(config.configParams.aws.publicKeyPath):
        warn(f"{config.configParams.aws.publicKeyPath} is not a falid ssh key")
        config.configParams.aws.publicKeyPath = ask("Path to your public ssh key: ")
    
    config.configParams.agent.apiKey = "00000000000000000000000000000000"
    apiKey = ask("🐶 Datadog API key - default [00000000000000000000000000000000]: ")
    if len(apiKey) == 32:
        config.configParams.agent.apiKey = apiKey
    
    config.configParams.agent.appKey = "0000000000000000000000000000000000000000"
    appKey = ask("🐶 Datadog APP key - default [0000000000000000000000000000000000000000]: ")
    if len(appKey) == 40:
        config.configParams.agent.appKey = appKey
    
    config.save_to_local_config()
    profile_path = Path.home().joinpath(profile_filename)
    debug("=========================================")
    with open(profile_path, 'r') as f:
        debug(f.read())
    debug("=========================================")
      
    

    


