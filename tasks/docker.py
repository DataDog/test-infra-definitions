from invoke.tasks import task
from .destroy import destroy
from .deploy import deploy
from . import doc
from typing import Optional
from invoke.context import Context
from . import tool
import pyperclip

scenario_name = "aws/dockervm"


@task(
    help={
        "install_agent": doc.install_agent,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
    }
)
def create_docker(
    ctx: Context,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_docker: Optional[bool] = False,
     agent_version: Optional[str] = None,
):
    """
    Create a docker environment.
    """
    full_stack_name = deploy(
        ctx,
        scenario_name,
        key_pair_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
        extra_flags={},
    )
    _show_connection_message(ctx, full_stack_name)


def _show_connection_message(ctx: Context, full_stack_name: str):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    connection = tool.Connection(outputs)
    host = connection.host
    user = connection.user

    command = (
        f'\nssh {user}@{host} --  \'echo "Successfully connected to VM" && exit\' \n'
        + f'docker context create pulumi-{host} --docker "host=ssh://{user}@{host}"\n'
        + f"docker --context pulumi-{host} container ls\n"
    )
    pyperclip.copy(command)
    print(
        f"If you want to use docker context, you can run the following commands which were copied in the clipboard\n\n{command}"
    )


@task(
    help={
        "stack_name": doc.stack_name,
    }
)
def destroy_docker(ctx: Context, stack_name: Optional[str] = None):
    """
    Destroy an environment created by invoke create_docker.
    """
    destroy(ctx, scenario_name, stack_name)
