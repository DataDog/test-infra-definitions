import os
import os.path
from pathlib import Path
import getpass
import pyperclip

from invoke import task
from invoke.context import Context

from .config import Config, get_full_profile_path
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
    config = Config()
    config.configParams = Config.Params()
    config.configParams.agent = Config.Params.Agent()
    config.configParams.aws = Config.Params.Aws()

    config.configParams.aws.keyPairName = getpass.getuser()
    keyPairName = ask(f"🔑 Key pair name - stored in AWS Sandbox, default [{config.configParams.aws.keyPairName}]: ")
    if len(keyPairName) > 0:
        config.configParams.aws.keyPairName = keyPairName

    checkKeyPair = ask("Did you create your SSH key on AWS and want me to check it is loaded on your ssh agent when creating manual environments or running e2e tests [Y/N]? Default No: ")
    if len(checkKeyPair) > 0:
        config.options = Config.Options()
        config.options.checkKeyPair = checkKeyPair.lower() == "y" or checkKeyPair.lower() == "yes"

    config.configParams.aws.publicKeyPath = ""
    while not os.path.isfile(config.configParams.aws.publicKeyPath):
        config.configParams.aws.publicKeyPath = Path.home().joinpath(".ssh", "id_ed25519.pub")
        publicKeyPath = ask(f"🔑 Path to your public ssh key, default [{config.configParams.aws.publicKeyPath}]: ")
        if len(publicKeyPath) > 0:
            config.configParams.aws.publicKeyPath = publicKeyPath
        if not os.path.isfile(config.configParams.aws.publicKeyPath):
            warn(f"{config.configParams.aws.publicKeyPath} is not a valid ssh key")
    
    config.configParams.agent.apiKey = ""
    while len(config.configParams.agent.apiKey) != 32:
        config.configParams.agent.apiKey = "00000000000000000000000000000000"
        apiKey = ask("🐶 Datadog API key - default [00000000000000000000000000000000]: ")
        if len(apiKey) > 0:
            config.configParams.agent.apiKey = apiKey
        if len(config.configParams.agent.apiKey) != 32:
            warn(f"Expecting API key of length 32, got {len(config.configParams.agent.apiKey)}")
        
    config.configParams.agent.appKey = ""
    while len(config.configParams.agent.appKey) != 40:
        config.configParams.agent.appKey = "0000000000000000000000000000000000000000"
        appKey = ask("🐶 Datadog APP key - default [0000000000000000000000000000000000000000]: ")
        if len(appKey) > 0:
            config.configParams.agent.appKey = appKey
        if len(config.configParams.agent.appKey) != 40:
            warn(f"Expecting APP key of length 40, got {len(config.configParams.agent.appKey)}")
    

    config.stackParams = {}

    config.save_to_local_config()
    cat_profile_command = f"cat {get_full_profile_path()}"
    pyperclip.copy(cat_profile_command)
    print(
        f"\nYou can run the following command to print your configuration: `{cat_profile_command}`. This command was copied to the clipboard\n"
    )
    
      
    

    


