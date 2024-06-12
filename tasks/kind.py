from typing import Optional

import pyperclip
from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task

from . import doc, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "aws/kind"


# TODO add dogstatsd and workload options
@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_agent_with_operator": doc.install_agent_with_operator,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "architecture": doc.architecture,
        "use_fakeintake": doc.fakeintake,
        "use_loadBalancer": doc.use_loadBalancer,
        "interactive": doc.interactive,
        "use_aws_vault": doc.use_aws_vault,
    }
)
def create_kind(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_agent_with_operator: Optional[bool] = False,
    agent_version: Optional[str] = None,
    architecture: Optional[str] = None,
    use_fakeintake: Optional[bool] = False,
    use_loadBalancer: Optional[bool] = False,
    interactive: Optional[bool] = True,
    use_aws_vault: Optional[bool] = True,
):
    """
    Create a kind environment.
    """

    extra_flags = {}
    extra_flags["ddinfra:osDescriptor"] = f"amazonlinuxecs::{_get_architecture(architecture)}"
    extra_flags["ddinfra:deployFakeintakeWithLoadBalancer"] = use_loadBalancer
    extra_flags["ddinfra:aws/defaultInstanceType"] = "t3.xlarge"

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        key_pair_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        install_agent_with_operator=install_agent_with_operator,
        agent_version=agent_version,
        use_fakeintake=use_fakeintake,
        extra_flags=extra_flags,
        use_aws_vault=use_aws_vault,
        app_key_required=True,
    )

    if interactive:
        tool.notify(ctx, "Your Kind environment is now created")

    _show_connection_message(ctx, full_stack_name, interactive)


def _show_connection_message(ctx: Context, full_stack_name: str, copy_to_clipboard: Optional[bool]):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    remoteHost = tool.RemoteHost("aws-kind", outputs)
    host = remoteHost.host
    user = remoteHost.user

    command = f"\nssh {user}@{host}"
    print(f"If you want to connect to the remote host, you can run the following command \n\n{command}")

    if copy_to_clipboard:
        input("Press a key to copy command to clipboard...")
        pyperclip.copy(command)


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "yes": doc.yes,
        "use_aws_vault": doc.use_aws_vault,
    }
)
def destroy_kind(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    yes: Optional[bool] = False,
    use_aws_vault: Optional[bool] = True,
):
    """
    Destroy an environment created by invoke create_docker.
    """
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
        use_aws_vault=use_aws_vault,
        force_yes=yes,
    )


def _get_architecture(architecture: Optional[str]) -> str:
    architectures = tool.get_architectures()
    if architecture is None:
        architecture = tool.get_default_architecture()
    if architecture.lower() not in architectures:
        raise Exit(f"The os family '{architecture}' is not supported. Possibles values are {', '.join(architectures)}")
    return architecture
