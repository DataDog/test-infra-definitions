from invoke.tasks import task
from pydantic import ValidationError
from .destroy import destroy
from .deploy import deploy
from . import config, doc
from typing import Optional
from invoke.context import Context
from invoke.exceptions import Exit
from . import tool
import pyperclip

scenario_name = "aws/ecs"


@task(
    help={
        "install_agent": doc.install_agent,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "use_fargate": doc.use_fargate,
        "linux_node_group": doc.linux_node_group,
        "linux_arm_node_group": doc.linux_arm_node_group,
        "bottlerocket_node_group": doc.bottlerocket_node_group,
        "windows_node_group": doc.windows_node_group,
    }
)
def create_ecs(
    ctx: Context,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
    use_fargate: bool = True,
    linux_node_group: bool = True,
    linux_arm_node_group: bool = False,
    bottlerocket_node_group: bool = False,
    windows_node_group: bool = False,
):
    """
    Create a new ECS environment.
    """
    extra_flags = {}
    extra_flags["ddinfra:aws/ecs/fargateCapacityProvider"] = use_fargate
    extra_flags["ddinfra:aws/ecs/linuxECSOptimizedNodeGroup"] = linux_node_group
    extra_flags["ddinfra:aws/ecs/linuxECSOptimizedARMNodeGroup"] = linux_arm_node_group
    extra_flags["ddinfra:aws/ecs/linuxBottlerocketNodeGroup"] = bottlerocket_node_group
    extra_flags["ddinfra:aws/ecs/windowsLTSCNodeGroup"] = windows_node_group

    full_stack_name = deploy(
        ctx,
        scenario_name,        
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
        extra_flags=extra_flags,
    )
    _show_connection_message(ctx, full_stack_name)


def _show_connection_message(ctx: Context, full_stack_name: str):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    cluster_name = outputs["ecs-cluster-name"]

    try:
        local_config = config.get_local_config()
    except ValidationError as e:
        raise Exit(f"Error in config {config.get_full_profile_path()}:{e}")

    command = f"{tool.get_aws_wrapper(local_config.get_aws().get_account())} -- aws ecs list-tasks --cluster {cluster_name}"
    pyperclip.copy(command)
    print(
        f"\nYou can run the following command to list tasks on the ECS cluster\n\n{command}\n\nThis command was copied to the clipboard\n"
    )


@task(
    help={
        "stack_name": doc.stack_name,
    }
)
def destroy_ecs(ctx: Context, stack_name: Optional[str] = None):
    """
    Destroy a ECS environment created with invoke create-ecs.
    """
    destroy(ctx, scenario_name, stack_name)
