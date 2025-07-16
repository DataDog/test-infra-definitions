from typing import Optional

from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic_core._pydantic_core import ValidationError

from tasks import config, doc
from tasks.config import get_full_profile_path
from tasks.deploy import deploy
from tasks.destroy import destroy

scenario_name = "gcp/openshiftvm"


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
    pull_secret_path: Optional[str] = None,
    use_nested_virtualization: Optional[bool] = True,
):
    """
    Create an OpenShift environment.
    """

    try:
        cfg = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path(config_path)}") from e

    # Use parameter if provided during invoke setup, otherwise use config
    if not pull_secret_path:
        pull_secret_path = cfg.get_gcp().pullSecretPath
        if not pull_secret_path:
            raise Exit(
                "pull_secret_path is required. Either use invoke.gcp.create-openshift -p <pull_secret_path> or configure it with 'invoke setup'"
            )

    extra_flags = {
        "scenario": scenario_name,
        "ddinfra:env": f"gcp/{cfg.get_gcp().account}",
        "ddinfra:gcp/defaultPublicKeyPath": cfg.get_gcp().publicKeyPath,
        "ddinfra:gcp/openshift/pullSecretPath": pull_secret_path,
        "ddinfra:gcp/enableNestedVirtualization": use_nested_virtualization,
        "ddinfra:gcp/defaultInstanceType": "n2-standard-8",
    }

    deploy(
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
    Destroy an environment created by invoke gcp.create-openshift.
    """
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
    )
