from invoke import task
from .config import get_config
import subprocess
import os
import invoke
from .stack import stack_vm, get_scenario_folder, get_stack_name


@task(help={
    'install_agent': "Install the Agent (default True).",
    'agent_version': "The version of the Agent for example '7.42.0~rc.1-1' or '6.39.0'",
})
def vm(ctx,
       install_agent=True,
       agent_version=""
       ):
    """
    Create a new virtual machine on the cloud.
    """
    config = get_config()
    aws_deploy(
        config,
        stack_vm,
        install_agent,
        f"ddagent:deploy={install_agent}",
        f"ddagent:version={agent_version}",
    )


def aws_deploy(config, scenario_name, install_agent, *args):
    api_key = ""
    if install_agent:
        api_key = os.getenv("DD_API_KEY")
        if api_key is None or len(api_key) != 32:
            raise invoke.Exit("Invalid API KEY. You must define the environment variable 'DD_API_KEY'. " +
                              "If you don't want an agent installation add '--no-install-agent'.")

    scenario_folder = get_scenario_folder(scenario_name)
    stack_name = get_stack_name(scenario_name)
    cmd_args = [
        "aws-vault", "exec", "sandbox-account-admin", "--",
        "pulumi", "up",
        "-c", "ddinfra:aws/defaultKeyPairName=" + config["aws_key_pair"],
        "-c", "ddinfra:env=aws/sandbox",
        "-c", "ddagent:apiKey=" + api_key,
        "-C", scenario_folder,
        "-s", stack_name]
    for arg in args:
        cmd_args.append("-c")
        cmd_args.append(arg)
    subprocess.call(cmd_args)
