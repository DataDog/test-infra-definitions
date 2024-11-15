from typing import Optional

from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic_core._pydantic_core import ValidationError

from tasks import config, doc
from tasks.deploy import deploy
from tasks.destroy import destroy
from tasks.tool import notify, show_connection_message

scenario_name = "localdocker/vm"
remote_hostname = "localdocker-vm"


@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "pipeline_id": doc.pipeline_id,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
        "debug": doc.debug,
        "use_fakeintake": doc.fakeintake,
        "interactive": doc.interactive,
    }
)
def create_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    pipeline_id: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    use_fakeintake: Optional[bool] = False,
    interactive: Optional[bool] = True,
) -> None:
    """
    Create a new virtual machine on local docker.
    """

    try:
        cfg = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {config.get_full_profile_path(config_path)}") from e

    if not cfg.get_local().publicKeyPath:
        raise Exit("The field `local.publicKeyPath` is required in the config file")

    extra_flags = {
        "ddinfra:local/defaultPublicKeyPath": cfg.get_gcp().publicKeyPath,
    }

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        stack_name=stack_name,
        pipeline_id=pipeline_id,
        install_agent=install_agent,
        agent_version=agent_version,
        debug=debug,
        extra_flags=extra_flags,
        use_fakeintake=use_fakeintake,
    )

    if interactive:
        notify(ctx, "Your VM is now created")

    show_connection_message(ctx, remote_hostname, full_stack_name, interactive)


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
    }
)
def destroy_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
):
    """
    Destroy a new virtual machine on aws.
    """
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
    )
