from invoke import task
from .deploy import deploy
from . import doc
from typing import Optional
from invoke.context import Context

@task(help={
    'install_agent': doc.install_agent,
    'agent_version': doc.agent_version,
})
def vm(ctx: Context,
       install_agent: bool =True,
       agent_version: Optional[str]= None
       ):
    """
    Create a new virtual machine on the cloud.
    """
    deploy(
        ctx, 
        "aws/vm", 
        install_agent=install_agent, 
        agent_version=agent_version)
