import os
from typing import Optional

import pyperclip
import yaml
from invoke.context import Context
from invoke.tasks import task

from . import doc, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "az/aks"


@task(
    help={
        "install_agent": doc.install_agent,
        "install_workload": doc.install_workload,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
    }
)
def create_aks(
    ctx: Context,
    debug: Optional[bool] = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_workload: Optional[bool] = True,
    agent_version: Optional[str] = None,
):
    """
    Create a new AKS environment. It lasts around 5 minutes.
    """

    extra_flags = {}
    extra_flags["ddinfra:env"] = "az/sandbox"

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

    _show_connection_message(ctx, full_stack_name)


def _show_connection_message(ctx: Context, full_stack_name: str):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    print(outputs)
    kubeconfig_output = yaml.safe_load(outputs["dd-Cluster-az-aks"]["kubeConfig"])
    kubeconfig_content = yaml.dump(kubeconfig_output)
    kubeconfig = f"{full_stack_name}-config.yaml"
    f = os.open(path=kubeconfig, flags=(os.O_WRONLY | os.O_CREAT | os.O_TRUNC), mode=0o600)
    with open(f, "w") as f:
        f.write(kubeconfig_content)

    command = f"KUBECONFIG={kubeconfig} kubectl get nodes"

    print(f"\nYou can run the following command to connect to the AKS cluster\n\n{command}\n")

    input("Press a key to copy command to clipboard...")
    pyperclip.copy(command)


@task(help={"stack_name": doc.stack_name, "yes": doc.yes})
def destroy_aks(ctx: Context, stack_name: Optional[str] = None, yes: Optional[bool] = False):
    """
    Destroy a AKS environment created with invoke create-aks.
    """
    destroy(ctx, scenario_name, stack_name, force_yes=yes, use_aws_vault=False)
