import os
from invoke.tasks import task
import yaml
from .destroy import destroy
from .deploy import deploy
from . import doc
from typing import Optional
from invoke.context import Context
from . import tool
import pyperclip

scenario_name = "aws/eks"


@task(
    help={
        "install_agent": doc.install_agent,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "linux_node_group": doc.linux_node_group,
        "linux_arm_node_group": doc.linux_arm_node_group,
        "bottlerocket_node_group": doc.bottlerocket_node_group,
        "windows_node_group": doc.windows_node_group,
    }
)
def create_eks(
    ctx: Context,
    debug: Optional[bool] = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
    linux_node_group: bool = True,
    linux_arm_node_group: bool = False,
    bottlerocket_node_group: bool = False,
    windows_node_group: bool = False,
):
    """
    Create a new EKS environment. It lasts around 20 minutes.
    """

    extra_flags = {}
    extra_flags["ddinfra:aws/eks/linuxNodeGroup"] = linux_node_group
    extra_flags["ddinfra:aws/eks/linuxARMNodeGroup"] = linux_arm_node_group
    extra_flags["ddinfra:aws/eks/linuxBottlerocketNodeGroup"] = bottlerocket_node_group
    extra_flags["ddinfra:aws/eks/windowsNodeGroup"] = windows_node_group

    full_stack_name = deploy(
        ctx,
        scenario_name,
        debug=debug,
        app_key_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
        extra_flags=extra_flags,
    )
    _show_connection_message(full_stack_name)


def _show_connection_message(full_stack_name: str):
    outputs = tool.get_stack_json_outputs(full_stack_name)
    kubeconfig = outputs["kubeconfig"]
    kubeconfig_content = yaml.dump(kubeconfig)
    config = f"{full_stack_name}-config.yaml"
    f = os.open(path=config, flags=(os.O_WRONLY | os.O_CREAT | os.O_TRUNC), mode=0o600)
    with open(f, "w") as f:
        f.write(kubeconfig_content)

    command = f"KUBECONFIG={config} aws-vault exec sandbox-account-admin -- kubectl get nodes"
    pyperclip.copy(command)
    print(
        f"\nYou can run the following command to connect to the EKS cluster\n\n{command}\n\nThis command was copied to the clipboard\n"
    )


@task(
    help={
        "stack_name": doc.stack_name,
    }
)
def destroy_eks(ctx: Context, stack_name: Optional[str] = None):
    """
    Destroy a EKS environment created with invoke create-eks.
    """
    destroy(ctx, scenario_name, stack_name)
