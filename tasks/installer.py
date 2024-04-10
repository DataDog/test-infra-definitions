from invoke.context import Context
from typing import Optional
from invoke.tasks import task
from . import doc

from .deploy import deploy
from .destroy import destroy

scenario_name = "aws/installer"

@task(
    help={
        "debug": doc.debug,
        "pipeline_id": doc.pipeline_id,
    }
)
def create_installer_lab(
    ctx: Context,
    debug: Optional[bool] = False,
    pipeline_id: Optional[str] = None,
):
    full_stack_name = deploy(
        ctx,
        scenario_name,
        stack_name="installer-lab",
        pipeline_id=pipeline_id,
        install_updater=True,
        debug=debug,
    )

    print(f"Installer lab created: {full_stack_name}")

@task(
    help={
        "yes": doc.yes,
    }
)
def destroy_installer_lab(
    ctx: Context,
    yes: Optional[bool] = False,
):
    destroy(
        ctx,
        scenario_name,
        stack="installer-lab",
        force_yes=yes
    )

    print("Installer lab destroyed")
