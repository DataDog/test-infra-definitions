from typing import Optional

from invoke.context import Context
from invoke.tasks import task

from . import doc
from . import config, tool
from .deploy import deploy
from .destroy import destroy

scenario_name = "aws/installer"


@task(
    help={
        "debug": doc.debug,
        "site": doc.site,
    }
)
def create_installer_lab(
    ctx: Context,
    config_path: Optional[str] = None,
    debug: Optional[bool] = False,
    remote: Optional[bool] = False,
    site: Optional[str] = "datad0g.com",
):
    cfg = config.get_local_config(config_path)
    aws_account = cfg.get_aws().get_account()
    flags = {
        "scenario": scenario_name,
        "ddinfra:env": "aws/" + aws_account,
        # TODO: change it to be a shared keypair in secret manager
        "ddinfra:aws/defaultKeyPairName": cfg.get_aws().keyPairName,
        # TODO: change it to be a shared api key in secret manager
        "ddagent:apiKey": cfg.get_installer().stagingApiKey,
    }
    stack_name = f"staging-installer-lab"
    if site != "datad0g.com":
        stack_name = f"prod-installer-lab"
    if remote:
        stack_name = tool.get_stack_name(stack_name, scenario_name)

    up_flags = ""

    for key, value in flags.items():
        up_flags += f" -c {key}={value}"
    if debug:
        up_flags += " --debug"

    aws_wrapper = tool.get_aws_wrapper(aws_account)
    # Connect to the bucket if needed
    if remote:
        bucket = cfg.get_installer().bucket
        ctx.run(f"{aws_wrapper} pulumi login s3://{bucket}")

    print("Stack init")
    # Create Stack
    ctx.run(f"pulumi stack init --no-select {stack_name}")

    print("Stack up")
    # Stack up
    cmd = f"pulumi up --yes -s {stack_name} {up_flags}"
    if not remote:
        cmd = tool.get_aws_wrapper(aws_account) + cmd

    ctx.run(cmd, pty=True)


@task(
    help={
        "yes": doc.yes,
        "site": doc.site,
    }
)
def destroy_installer_lab(
    ctx: Context,
    config_path: Optional[str] = None,
    remote: Optional[bool] = False,
    yes: Optional[bool] = False,
    site: Optional[str] = "datad0g.com",
):
    stack_name = f"staging-installer-lab"
    if site != "datad0g.com":
        stack_name = f"prod-installer-lab"
    if remote:
        stack_name = tool.get_stack_name(stack_name, scenario_name)

    cfg = config.get_local_config(config_path)
    aws_account = cfg.get_aws().get_account()

    # Connect to the bucket if needed
    if remote:
        bucket = cfg.get_installer().bucket
        aws_wrapper = tool.get_aws_wrapper(aws_account)
        ctx.run(f"{aws_wrapper} pulumi login s3://{bucket}")

    force_destroy = "--yes --skip-preview" if yes else ""
    cmd = f"pulumi destroy --remove -s {stack_name} {force_destroy}"
    if not remote:
        cmd = tool.get_aws_wrapper(aws_account) + cmd

    ctx.run(cmd, pty=True)
