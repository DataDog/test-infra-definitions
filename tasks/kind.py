from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from . import doc


# TODO add dogstatsd and workload options
@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_agent_with_operator": doc.install_agent_with_operator,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "architecture": doc.architecture,
        "use_fakeintake": doc.fakeintake,
        "use_loadBalancer": doc.use_loadBalancer,
        "interactive": doc.interactive,
    }
)
def create_kind(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_agent_with_operator: Optional[bool] = False,
    agent_version: Optional[str] = None,
    architecture: Optional[str] = None,
    use_fakeintake: Optional[bool] = False,
    use_loadBalancer: Optional[bool] = False,
    interactive: Optional[bool] = True,
):
    print('This command is deprecated, please use `aws.create-kind` instead')
    print("Running `aws.create-kind`...")
    from tasks.aws.kind import create_kind as create_kind_aws

    create_kind_aws(
        ctx,
        config_path,
        key_pair_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        install_agent_with_operator=install_agent_with_operator,
        agent_version=agent_version,
        use_fakeintake=use_fakeintake,
        extra_flags=extra_flags,
        app_key_required=True,
    )


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "yes": doc.yes,
    }
)
def destroy_kind(
    ctx: Context,
    config_path: Optional[str] = None,
    stack_name: Optional[str] = None,
    yes: Optional[bool] = False,
):
    print('This command is deprecated, please use `aws.destroy-kind` instead')
    print("Running `aws.destroy-kind`...")
    from tasks.aws.kind import destroy_kind as destroy_kind_aws

    destroy_kind_aws(ctx, config_path, stack_name, yes)
