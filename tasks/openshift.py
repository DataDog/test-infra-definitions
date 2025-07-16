import os
from typing import Optional

from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task

from tasks import doc
from tasks.deploy import deploy
from tasks.destroy import destroy

scenario_name = "gcp/openshiftvm"


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
        "pull_secret_path": doc.pull_secret_path,
    }
)
def create_openshift(
    ctx: Context,
    stack_name: Optional[str] = None,
    pull_secret_path: Optional[str] = None,
):
    """
    Create an OpenShift environment.
    """

    extra_flags = {
        "ddinfra:openShiftPullSecretPath": pull_secret_path,
    }

    full_stack_name = deploy(
        ctx,
        scenario_name,
        stack_name=stack_name,
        extra_flags=extra_flags,
    )


@task(
    help={
        "config_path": doc.config_path,
        "stack_name": doc.stack_name,
    }
)
def destroy_openshift(
    ctx: Context,
    scenario_name: str,
    stack_name: Optional[str] = None,
    pull_secret_path: Optional[str] = None,
):
    """
    Destroy an environment created by invoke openshift.create-openshift.
    """
    destroy(
        ctx,
        scenario_name=scenario_name
        stack=stack_name,
        pull_secret_path=pull_secret_path,
    ) 