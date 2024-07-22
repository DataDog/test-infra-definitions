from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from . import doc


@task(
    help={
        "install_agent": doc.install_agent,
        "install_workload": doc.install_workload,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
    }
)
def create_aks(
    ctx: Context,
    debug: Optional[bool] = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_workload: Optional[bool] = True,
    agent_version: Optional[str] = None,
):
    print('This command is deprecated, please use `az.create-aks` instead')
    print("Running `az.create-aks`...")
    from tasks.azure.aks import create_aks as create_aks_azure

    create_aks_azure(
        ctx,
        debug=debug,
        stack_name=stack_name,
        install_agent=install_agent,
        install_workload=install_workload,
        agent_version=agent_version,
    )


@task(help={"stack_name": doc.stack_name, "yes": doc.yes})
def destroy_aks(ctx: Context, stack_name: Optional[str] = None, yes: Optional[bool] = False):
    print('This command is deprecated, please use `az.destroy-aks` instead')
    print("Running `az.destroy-aks`...")
    from tasks.azure.aks import destroy_aks as destroy_aks_azure

    destroy_aks_azure(ctx, stack_name=stack_name, yes=yes)
