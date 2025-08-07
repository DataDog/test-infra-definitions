from typing import Optional

from invoke import task
from invoke.context import Context

from .deploy import deploy
from ..destroy import destroy
from tasks import doc


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "install_agent": doc.install_agent,
        "agent_version": doc.agent_version,
    }
)
def create_asg(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
) -> None:
    """
    Create a new autoscaling group.
    """
    extra_flags = {}
    if install_agent:
        extra_flags["ddinfra:agent"] = "true"
    if agent_version:
        extra_flags["ddagent:version"] = agent_version

    deploy(
        ctx,
        "aws/asg",
        config_path=config_path,
        stack_name=stack_name,
        extra_flags=extra_flags,
    )


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
    }
)
def destroy_asg(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
) -> None:
    """
    Destroy an existing autoscaling group.
    """
    destroy(
        ctx,
        scenario_name="aws/asg",
        config_path=config_path,
        stack=stack_name,
    ) 