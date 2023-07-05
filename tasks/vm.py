from invoke.tasks import task

from .destroy import destroy
from .deploy import deploy
from . import doc
from typing import Optional, Tuple
from invoke.context import Context
from invoke.exceptions import Exit
from . import tool
import pyperclip

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
) -> None:
    """
    Create a new virtual machine on the cloud.
    """

    extra_flags = {}
    os_family, os_arch = _get_os_information(ctx, os_family, ami_id)
    extra_flags["ddinfra:osFamily"] = os_family
    if ami_id is not None:
        extra_flags["ddinfra:osArchitecture"] = os_arch
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
    _show_connection_message(full_stack_name)


def _show_connection_message(full_stack_name: str):
    outputs = tool.get_stack_json_outputs(full_stack_name)
    connection = tool.Connection(outputs)
    host = connection.host
    user = connection.user

    command = f"ssh {user}@{host}"
    pyperclip.copy(command)
    print(
        f"\nYou can run the following command to connect to the host `{command}`. This command was copied to the clipboard\n"
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
    destroy(ctx, scenario_name, stack_name)


def _get_os_family(os_family: Optional[str]) -> str:
    os_families = tool.get_os_families()
    if os_family is None:
        os_family = tool.get_default_os_family()
    if os_family.lower() not in os_families:
        raise Exit(
            f"The os family '{os_family}' is not supported. Possibles values are {', '.join(os_families)}"
        )
    return os_family


def _get_os_information(
    ctx: Context,
    os_family: Optional[str],
    ami_id: Optional[str]
) -> Tuple[str, Optional[str]]:
    family, architecture = os_family, None
    if ami_id is not None:
        image = tool.get_image_description(ctx, ami_id)
        if family is None:  # Try to guess the distribution
            os_families = tool.get_os_families()
            try:
                if "Description" in image:
                    family = next(
                        os
                        for os in os_families
                        if os in image["Description"].lower().replace(" ", "")
                    )
                else: # If the image has no description, check with name
                    family = next(
                        os
                        for os in os_families
                        if os in image["Name"].lower().replace(" ", "")
                    )

            except StopIteration:
                raise Exit(
                    f"We failed to guess the family of your AMI ID. Please provide it with option -o"
                )
        architecture = image["Architecture"]
    else:
        family = _get_os_family(os_family)
    return (family, architecture)
