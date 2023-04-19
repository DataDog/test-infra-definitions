from invoke import task
from .deploy import deploy
from . import doc
from typing import Optional
from invoke.context import Context


@task(
    help={
        "install_agent": doc.install_agent,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
    }
)
def vm(
    ctx: Context,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = None,
    agent_version: Optional[str] = None,
):
    """
    Create a new virtual machine on the cloud.
    """
    deploy(
        ctx,
        "aws/vm",
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
    )
