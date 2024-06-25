import getpass
import os
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
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_updater": doc.install_updater,
        "pipeline_id": doc.pipeline_id,
        "agent_version": doc.agent_version,
        "stack_name": doc.stack_name,
        "debug": doc.debug,
        "os_family": doc.os_family,
        "use_fakeintake": doc.fakeintake,
        "use_loadBalancer": doc.use_loadBalancer,
        "ami_id": doc.ami_id,
        "architecture": doc.architecture,
        "interactive": doc.interactive,
        "use_aws_vault": doc.use_aws_vault,
        "instance_type": doc.instance_type,
        "no_verify": doc.no_verify,
        "ssh_user": doc.ssh_user,
        "os_version": doc.os_version,
    }
)
def create_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    pipeline_id: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_updater: Optional[bool] = False,
    agent_version: Optional[str] = None,
    debug: Optional[bool] = False,
    os_family: Optional[str] = None,
    os_version: Optional[str] = None,
    use_fakeintake: Optional[bool] = False,
    use_loadBalancer: Optional[bool] = False,
    ami_id: Optional[str] = None,
    architecture: Optional[str] = None,
    use_aws_vault: Optional[bool] = True,
    interactive: Optional[bool] = True,
    instance_type: Optional[str] = None,
    no_verify: Optional[bool] = False,
    ssh_user: Optional[str] = None,
) -> None:
    """
    Create a new virtual machine on the cloud.
    """

    extra_flags = {}
    os_family, os_arch = _get_os_information(ctx, os_family, architecture, ami_id)
    deploy_job = None if no_verify else tool.get_deploy_job(os_family, os_arch, agent_version)
    extra_flags["ddinfra:osDescriptor"] = f"{os_family}:{os_version if os_version else ''}:{os_arch}"
    extra_flags["ddinfra:deployFakeintakeWithLoadBalancer"] = use_loadBalancer

    if ami_id is not None:
        extra_flags["ddinfra:osImageID"] = ami_id

    if use_fakeintake and not install_agent:
        print(
            "[WARNING] It is currently not possible to deploy a VM with fakeintake and without agent. Your VM will start without fakeintake."
        )
    if instance_type is not None:
        if architecture is None or architecture.lower() == tool.get_default_architecture():
            extra_flags["ddinfra:aws/defaultInstanceType"] = instance_type
        else:
            extra_flags["ddinfra:aws/defaultARMInstanceType"] = instance_type

    if ssh_user:
        extra_flags["ddinfra:sshUser"] = ssh_user

    full_stack_name = deploy(
        ctx,
        scenario_name,
        config_path,
        key_pair_required=True,
        public_key_required=(os_family.lower() == "windows"),
        stack_name=stack_name,
        pipeline_id=pipeline_id,
        install_agent=install_agent,
        install_updater=install_updater,
        agent_version=agent_version,
        debug=debug,
        extra_flags=extra_flags,
        use_fakeintake=use_fakeintake,
        use_aws_vault=use_aws_vault,
        deploy_job=deploy_job,
    )

    if interactive:
        tool.notify(ctx, "Your VM is now created")

    _show_connection_message(ctx, full_stack_name, interactive)


def _show_connection_message(ctx: Context, full_stack_name: str, copy_to_clipboard: Optional[bool] = True):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    remoteHost = tool.RemoteHost("aws-vm", outputs)
    host = remoteHost.host
    user = remoteHost.user

    command = f"ssh {user}@{host}"

    print(f"\nYou can run the following command to connect to the host `{command}`.\n")
    if copy_to_clipboard:
        input("Press a key to copy command to clipboard...")
        pyperclip.copy(command)


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "yes": doc.yes,
        "use_aws_vault": doc.use_aws_vault,
        "clean_known_hosts": doc.clean_known_hosts,
    }
)
def destroy_vm(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    yes: Optional[bool] = False,
    use_aws_vault: Optional[bool] = True,
    clean_known_hosts: Optional[bool] = True,
):
    """
    Destroy a new virtual machine on the cloud.
    """
    host = _get_host(ctx, stack_name)
    destroy(
        ctx,
        scenario_name=scenario_name,
        config_path=config_path,
        stack=stack_name,
        use_aws_vault=use_aws_vault,
        force_yes=yes,
    )
    if clean_known_hosts:
        _clean_known_hosts(host)


def _get_host(ctx: Context, stack_name: Optional[str] = None) -> str:
    """
    Get the host of the VM.
    """
    full_stack_name = tool.get_stack_name(stack_name, scenario_name)
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    remoteHost = tool.RemoteHost("aws-vm", outputs)
    return remoteHost.host


def _clean_known_hosts(host: str) -> None:
    """
    Remove the host from the known_hosts file.
    """
    home = os.environ.get("HOME", f"/Users/{getpass.getuser()}")
    with open(f"{home}/.ssh/known_hosts") as f:
        lines = f.readlines()
    filtered_lines = [line for line in lines if not line.startswith(host)]
    with open(f"{home}/.ssh/known_hosts", "w") as f:
        f.writelines(filtered_lines)


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
                if "Description" in image:
                    image_info = image["Description"]
                else:
                    image_info = image["Name"]
                image_info = image_info.lower().replace(" ", "")
                family = next(os for os in os_families if os in image_info)

            except StopIteration:
                raise Exit("We failed to guess the family of your AMI ID. Please provide it with option -o")
        architecture = image["Architecture"]
        if arch is not None and architecture != arch:
            raise Exit(f"The provided architecture is {arch} but the image is {architecture}.")
    else:
        family = _get_os_family(os_family)
        architecture = _get_architecture(arch)
    return (family, architecture)
