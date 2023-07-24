from typing import Optional, Tuple

import pyperclip
from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task

from . import doc, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "aws/vm"


@task(
    help={
        "install_agent": doc.install_agent,
        "pipeline_id": doc.pipeline_id,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
        "debug": doc.debug,
        "os_family": doc.os_family,
        "use_fakeintake": doc.fakeintake,
        "ami_id": doc.ami_id,
        "architecture": doc.architecture,
    }
)
def create_vm(
    ctx: Context,
    stack_name: Optional[str] = None,
    pipeline_id: Optional[str] = None,
    install_agent: Optional[bool] = True,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    os_family: Optional[str] = None,
    use_fakeintake: Optional[bool] = True,
    ami_id: Optional[str] = None,
    architecture: Optional[str] = None,
) -> None:
    """
    Create a new virtual machine on the cloud.
    """

    extra_flags = {}
    os_family, os_arch = _get_os_information(ctx, os_family, architecture, ami_id)
    extra_flags["ddinfra:osFamily"] = os_family

    if os_arch is not None:
        extra_flags["ddinfra:osArchitecture"] = os_arch

    if ami_id is not None:
        extra_flags["ddinfra:osAmiId"] = ami_id

    full_stack_name = deploy(
        ctx,
        scenario_name,
        key_pair_required=True,
        public_key_required=(os_family.lower() == "windows"),
        stack_name=stack_name,
        pipeline_id=pipeline_id,
        install_agent=install_agent,
        agent_version=agent_version,
        debug=debug,
        extra_flags=extra_flags,
        use_fakeintake=use_fakeintake,
    )
    _show_connection_message(ctx, full_stack_name)


def _show_connection_message(ctx: Context, full_stack_name: str):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    connection = tool.Connection(outputs)
    host = connection.host
    user = connection.user

    command = f"ssh {user}@{host}"
    pyperclip.copy(command)
    print(
        f"\nYou can run the following command to connect to the host `{command}`. This command was copied to the clipboard\n"
    )


@task(help={"stack_name": doc.stack_name, "yes": doc.yes})
def destroy_vm(ctx: Context, stack_name: Optional[str] = None, yes: Optional[bool] = False):
    """
    Destroy a new virtual machine on the cloud.
    """
    destroy(ctx, scenario_name, stack_name, force_yes=yes)


def _get_os_family(os_family: Optional[str]) -> str:
    os_families = tool.get_os_families()
    if os_family is None:
        os_family = tool.get_default_os_family()
    if os_family.lower() not in os_families:
        raise Exit(f"The os family '{os_family}' is not supported. Possibles values are {', '.join(os_families)}")
    return os_family


def _get_architecture(architecture: Optional[str]) -> str:
    architectures = tool.get_architectures()
    if architecture is None:
        architecture = tool.get_default_architecture()
    if architecture.lower() not in architectures:
        raise Exit(f"The os family '{architecture}' is not supported. Possibles values are {', '.join(architectures)}")
    return architecture


def _get_os_information(
    ctx: Context, os_family: Optional[str], arch: Optional[str], ami_id: Optional[str]
) -> Tuple[str, Optional[str]]:
    family, architecture = os_family, None
    if ami_id is not None:
        image = tool.get_image_description(ctx, ami_id)
        if family is None:  # Try to guess the distribution
            os_families = tool.get_os_families()
            try:
                family = next(os for os in os_families if os in image["Description"].lower().replace(" ", ""))
            except StopIteration:
                raise Exit("We failed to guess the family of your AMI ID. Please provide it with option -o")
        architecture = image["Architecture"]
        if arch is not None and architecture != arch:
            raise Exit(f"The provided architecture is {arch} but the image is {architecture}.")
    else:
        family = _get_os_family(os_family)
        architecture = _get_architecture(arch)
    return (family, architecture)
