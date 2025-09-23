from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from tasks import config, doc
from tasks.config import get_full_profile_path
from tasks.deploy import deploy
from tasks.destroy import destroy
from invoke.exceptions import Exit
from pydantic_core._pydantic_core import ValidationError


scenario_name = "gcp/asg"


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
    account: Optional[str] = None,
) -> None:
    """
    Create a new GCP VM group (ASG-like) deployment.
    """

    try:
        cfg = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path(config_path)}") from e

    extra_flags = {
        "ddinfra:env": f"gcp/{account if account else cfg.get_gcp().account}",
        "ddinfra:gcp/defaultPublicKeyPath": cfg.get_gcp().publicKeyPath,
    }
    if agent_version:
        extra_flags["ddagent:version"] = agent_version

    deploy(
        ctx,
        scenario_name,
        config_path=config_path,
        stack_name=stack_name,
        install_agent=install_agent,
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
    Destroy an existing GCP VM group (ASG-like) deployment.
    """

    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
    )


