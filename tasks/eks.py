import os
from invoke import task
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
        "eks_linux_node_group": doc.linux_node_group,
        "eks_linux_arm_node_group": doc.linux_arm_node_group,
        "eks_bottlerocket_node_group": doc.bottlerocket_node_group,
        "eks_windows_node_group": doc.windows_node_group,
    }
)
def create_eks(
    ctx: Context,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = False,
    agent_version: Optional[str] = None,
    eks_linux_node_group: bool = True,
    eks_linux_arm_node_group: bool = False,
    eks_bottlerocket_node_group: bool = False,
    eks_windows_node_group: bool = False,
):
    """
    Create a new EKS environment. It lasts around 20 minutes.
    """

    extra_flags = {}
    extra_flags["ddinfra:aws/eks/linuxNodeGroup"] = eks_linux_node_group
    extra_flags["ddinfra:aws/eks/linuxARMNodeGroup"] = eks_linux_arm_node_group
    extra_flags[
        "ddinfra:aws/eks/linuxBottlerocketNodeGroup"
    ] = eks_bottlerocket_node_group
    extra_flags["ddinfra:aws/eks/windowsNodeGroup"] = eks_windows_node_group

    full_stack_name = deploy(
        ctx,
        scenario_name,
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

    f = os.open(
        path="config.yaml", flags=(os.O_WRONLY | os.O_CREAT | os.O_TRUNC), mode=0o600
    )
    with open(f, "w") as f:
        f.write(kubeconfig_content)

    command = "KUBECONFIG=config.yaml aws-vault exec sandbox-account-admin -- kubectl get nodes"
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
    destroy(scenario_name, stack_name)
