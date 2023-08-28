from typing import Optional

import pyperclip
from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task

from . import doc, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "aws/dockervm"


@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "architecture": doc.architecture,
    }
)
def create_docker(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
    architecture: Optional[str] = None,
):
    """
    Create a docker environment.
    """

    extra_flags = {}
    extra_flags["ddinfra:osArchitecture"] = _get_architecture(architecture)

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        key_pair_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
        extra_flags=extra_flags,
    )
    _show_connection_message(ctx, full_stack_name)


def _show_connection_message(ctx: Context, full_stack_name: str):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    connection = tool.Connection(outputs)
    host = connection.host
    user = connection.user

    command = (
        f"\nssh {user}@{host} --  'echo \"Successfully connected to VM\" && exit' \n"
        + f'docker context create pulumi-{host} --docker "host=ssh://{user}@{host}"\n'
        + f"docker --context pulumi-{host} container ls\n"
    )
    pyperclip.copy(command)
    print(
        f"If you want to use docker context, you can run the following commands which were copied in the clipboard\n\n{command}"
    )


@task(help={"stack_name": doc.stack_name, "yes": doc.yes})
def destroy_docker(ctx: Context, stack_name: Optional[str] = None, yes: Optional[bool] = False):
    """
    Destroy an environment created by invoke create_docker.
    """
    destroy(ctx, scenario_name, stack_name, force_yes=yes)


def _get_architecture(architecture: Optional[str]) -> str:
    architectures = tool.get_architectures()
    if architecture is None:
        architecture = tool.get_default_architecture()
    if architecture.lower() not in architectures:
        raise Exit(f"The os family '{architecture}' is not supported. Possibles values are {', '.join(architectures)}")
    return architecture
