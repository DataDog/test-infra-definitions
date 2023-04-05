from invoke import task
from .deploy import deploy
from . import doc

@task(help={
    'install_agent': doc.install_agent,
    'agent_version': doc.agent_version,
})
def vm(ctx,
       install_agent=True,
       agent_version=""
       ):
    """
    Create a new virtual machine on the cloud.
    """
    deploy(
        ctx, 
        "aws/vm", 
        install_agent=install_agent, 
        agent_version=agent_version)
