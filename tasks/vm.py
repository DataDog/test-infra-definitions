from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from . import doc


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
    interactive: Optional[bool] = True,
    instance_type: Optional[str] = None,
    no_verify: Optional[bool] = False,
    ssh_user: Optional[str] = None,
) -> None:
    from tasks.aws.vm import create_vm as create_vm_aws

    print('This command is deprecated, please use `aws.create-vm` instead')
    print("Running `aws.create-vm`...")
    create_vm_aws(
        ctx,
        config_path,
        stack_name,
        pipeline_id,
        install_agent,
        install_updater,
        agent_version,
        debug,
        os_family,
        os_version,
        use_fakeintake,
        use_loadBalancer,
        ami_id,
        architecture,
        interactive,
        instance_type,
        no_verify,
        ssh_user,
    )


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
    from tasks.aws.vm import destroy_vm as destroy_vm_aws

    print('This command is deprecated, please use `aws.destroy-vm` instead')
    print("Running `aws.destroy-vm`...")
    destroy_vm_aws(ctx, config_path, stack_name, yes, clean_known_hosts)
