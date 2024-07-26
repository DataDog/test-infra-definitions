from typing import Optional

from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic_core._pydantic_core import ValidationError

from tasks import config, doc, tool
from tasks.config import get_full_profile_path
from tasks.deploy import deploy
from tasks.destroy import destroy
from tasks.tool import clean_known_hosts as clean_known_hosts_func
from tasks.tool import get_host, show_connection_message

scenario_name = "az/vm"
remote_hostname = "az-vm"


@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_updater": doc.install_updater,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
        "debug": doc.debug,
        "interactive": doc.interactive,
        "ssh_user": doc.ssh_user,
    }
)
def create_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_updater: Optional[bool] = False,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    interactive: Optional[bool] = True,
    ssh_user: Optional[str] = None,
    account: Optional[str] = None,
) -> None:
    """
    Create a new virtual machine on azure.
    """

    try:
        cfg = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {get_full_profile_path(config_path)}") from e

    if not cfg.get_azure().publicKeyPath:
        raise Exit("The field `azure.publicKeyPath` is required in the config file")

    extra_flags = {
        "ddinfra:env": f"az/{account if account else cfg.get_azure().account}",
        "ddinfra:az/defaultPublicKeyPath": cfg.get_azure().publicKeyPath,
    }

    if ssh_user:
        extra_flags["ddinfra:sshUser"] = ssh_user

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        stack_name=stack_name,
        install_agent=install_agent,
        install_updater=install_updater,
        agent_version=agent_version,
        debug=debug,
        extra_flags=extra_flags,
    )

    if interactive:
        tool.notify(ctx, "Your VM is now created")

    show_connection_message(ctx, remote_hostname, full_stack_name, interactive)


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "yes": doc.yes,
        "clean_known_hosts": doc.clean_known_hosts,
    }
)
def destroy_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    yes: Optional[bool] = False,
    clean_known_hosts: Optional[bool] = True,
):
    """
    Destroy a new virtual machine on azure.
    """
    host = get_host(ctx, remote_hostname, scenario_name, stack_name)
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
        force_yes=yes,
    )
    if clean_known_hosts:
        clean_known_hosts_func(host)
