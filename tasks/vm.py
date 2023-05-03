from invoke import task

from .destroy import destroy
from .deploy import deploy
from . import doc
from typing import Optional
from invoke.context import Context
from . import tool
import invoke

scenario_name = "aws/vm"


@task(
    help={
        "install_agent": doc.install_agent,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
        "debug": doc.debug,
        "os_family": doc.os_family,
    }
)
def create_vm(
    ctx: Context,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = False,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    os_family: Optional[str] = None,
):
    """
    Create a new virtual machine on the cloud.
    """

    extra_flags = {}
    os_family = _get_os_family(os_family)
    extra_flags["ddinfra:osFamily"] = os_family

    deploy(
        ctx,
        scenario_name,
        key_pair_required=True,
        public_key_required=(os_family.lower() == "windows"),
        stack_name=stack_name,
        install_agent=install_agent,
        agent_version=agent_version,
        debug=debug
        extra_flags=extra_flags,
    )


@task(
    help={
        "stack_name": doc.stack_name,
    }
)
def destroy_vm(ctx: Context, stack_name: Optional[str] = None):
    """
    Destroy a new virtual machine on the cloud.
    """
    destroy(scenario_name, stack_name)


def _get_os_family(os_family: Optional[str]) -> str:
    os_families = tool.get_os_families()
    if os_family is None:
        os_family = tool.get_default_os_family()
    if os_family.lower() not in os_families:
        raise invoke.Exit(
            f"The os family '{os_family}' is not supported. Possibles values are {', '.join(os_families)}"
        )
    return os_family
