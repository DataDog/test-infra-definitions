import os
from typing import Optional

import pyperclip
import yaml
from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic import ValidationError

from . import config, doc, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "az/aks"


@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_workload": doc.install_workload,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
    }
)
def create_aks(
    ctx: Context,
    config_path: Optional[str] = None,
    debug: Optional[bool] = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_workload: Optional[bool] = True,
    agent_version: Optional[str] = None,
):
    """
    Create a new AKS environment. It lasts around 20 minutes.
    """

    extra_flags = {}
    extra_flags["ddinfra:env"] = "az/sandbox"
    extra_flags["ddinfra:kubernetesVersion"] = "1.27.7"  # TODO: remove this line when pulumi use alias minor version.

    full_stack_name = deploy(
        ctx,
        scenario_name,
        debug=debug,
        app_key_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        install_workload=install_workload,
        agent_version=agent_version,
        extra_flags=extra_flags,
        use_aws_vault=False,
    )

    tool.notify(ctx, "Your AKS cluster is now created")

    _show_connection_message(ctx, full_stack_name, config_path)


def _show_connection_message(ctx: Context, full_stack_name: str, config_path: Optional[str]):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    kubeconfig_output = outputs["kubeconfig"]
    kubeconfig_content = yaml.dump(kubeconfig_output)
    kubeconfig = f"{full_stack_name}-config.yaml"
    f = os.open(path=kubeconfig, flags=(os.O_WRONLY | os.O_CREAT | os.O_TRUNC), mode=0o600)
    with open(f, "w") as f:
        f.write(kubeconfig_content)

    try:
        local_config = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {config.get_full_profile_path(config_path)}:{e}")

    command = f"KUBECONFIG={kubeconfig} {tool.get_aws_wrapper(local_config.get_aws().get_account())} kubectl get nodes"

    print(f"\nYou can run the following command to connect to the AKS cluster\n\n{command}\n")

    input("Press a key to copy command to clipboard...")
    pyperclip.copy(command)


@task(help={"stack_name": doc.stack_name, "yes": doc.yes})
def destroy_aks(ctx: Context, stack_name: Optional[str] = None, yes: Optional[bool] = False):
    """
    Destroy a AKS environment created with invoke create-aks.
    """
    destroy(ctx, scenario_name, stack_name, force_yes=yes)
