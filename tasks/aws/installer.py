from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from tasks import doc
from tasks.aws.deploy import deploy
from tasks.destroy import destroy

scenario_name = "aws/installer"


@task(
    help={
        "debug": doc.debug,
        "pipeline_id": doc.pipeline_id,
        "site": doc.site,
    }
)
def create_installer_lab(
    ctx: Context,
    debug: Optional[bool] = False,
    pipeline_id: Optional[str] = None,
    site: Optional[str] = "datad0g.com",
):
    full_stack_name = deploy(
        ctx,
        scenario_name,
        stack_name="installer-lab",
        pipeline_id=pipeline_id,
        install_updater=True,
        debug=debug,
        extra_flags={"ddagent:site": site},
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
    destroy(ctx, scenario_name=scenario_name, stack="installer-lab", force_yes=yes)

    print("Installer lab destroyed")
