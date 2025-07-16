from typing import Optional

from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic_core._pydantic_core import ValidationError

from tasks import config, doc
from tasks.config import get_full_profile_path
from tasks.deploy import deploy
from tasks.destroy import destroy

scenario_name = "local/openshiftvm"


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "pull_secret_path": doc.pull_secret_path,
    }
)
def create_openshift(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    interactive: Optional[bool] = True,
    pull_secret_path: Optional[str] = None,
):
    """
    Create a local OpenShift environment.
    """

    try:
        cfg = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path(config_path)}") from e

    # Use parameter if provided during invoke setup, otherwise use config
    if not pull_secret_path:
        pull_secret_path = cfg.get_local().pullSecretPath
        if not pull_secret_path:
            raise Exit("pull_secret_path is required. Either use invoke.local.create-openshift -p <pull_secret_path> or configure it with 'invoke setup'")

    extra_flags = {
        "scenario": scenario_name,
        "ddinfra:env": "local",
        "ddinfra:local/defaultPublicKeyPath": cfg.get_local().publicKeyPath,
        "ddinfra:local/openshift/pullSecretPath": pull_secret_path,
    }

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        stack_name=stack_name,
        install_agent=False,
        extra_flags=extra_flags,
    )


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
    }
)
def destroy_openshift(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
):
    """
    Destroy an environment created by invoke local.create-openshift.
    """
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
    ) 