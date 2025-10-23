import json
import os
from typing import Optional

import pyperclip
import yaml
from invoke.context import Context
from invoke.exceptions import Exit
from invoke.tasks import task
from pydantic import ValidationError

from tasks import config, doc, tool
from tasks.aws import doc as aws_doc
from tasks.aws.common import get_aws_wrapper
from tasks.aws.deploy import deploy
from tasks.destroy import destroy

scenario_name = "aws/benchmarkeks"


@task(
    help={
        "config_path": doc.config_path,
        "install_agent": doc.install_agent,
        "install_workload": doc.install_workload,
        "install_argorollout": doc.install_argorollout,
        "agent_version": doc.container_agent_version,
        "stack_name": doc.stack_name,
        "linux_node_group": doc.linux_node_group,
        "linux_arm_node_group": doc.linux_arm_node_group,
        "bottlerocket_node_group": doc.bottlerocket_node_group,
        "windows_node_group": doc.windows_node_group,
        "instance_type": aws_doc.instance_type,
        "full_image_path": doc.full_image_path,
        "cluster_agent_full_image_path": doc.cluster_agent_full_image_path,
        "baseline_version": doc.baseline_version,
        "baseline_full_image_path": doc.baseline_full_image_path,
        "baseline_cluster_agent_version": doc.baseline_cluster_agent_version,
        "baseline_cluster_agent_full_image_path": doc.baseline_cluster_agent_full_image_path,
        "comparison_version": doc.comparison_version,
        "comparison_full_image_path": doc.comparison_full_image_path,
        "comparison_cluster_agent_version": doc.comparison_cluster_agent_version,
        "comparison_cluster_agent_full_image_path": doc.comparison_cluster_agent_full_image_path,
        "agent_flavor": doc.agent_flavor,
        "helm_config": doc.helm_config,
        "local_chart_path": doc.local_chart_path,
    }
)
def create_benchmarkeks(
    ctx: Context,
    config_path: Optional[str] = None,
    debug: Optional[bool] = False,
    stack_name: Optional[str] = None,
    install_agent: Optional[bool] = True,
    install_workload: Optional[bool] = True,
    install_argorollout: Optional[bool] = False,
    agent_version: Optional[str] = None,
    linux_node_group: bool = True,
    linux_arm_node_group: bool = False,
    bottlerocket_node_group: bool = True,
    windows_node_group: bool = False,
    instance_type: Optional[str] = None,
    full_image_path: Optional[str] = None,
    cluster_agent_full_image_path: Optional[str] = None,
    baseline_version: Optional[str] = None,
    baseline_full_image_path: Optional[str] = None,
    baseline_cluster_agent_version: Optional[str] = None,
    baseline_cluster_agent_full_image_path: Optional[str] = None,
    comparison_version: Optional[str] = None,
    comparison_full_image_path: Optional[str] = None,
    comparison_cluster_agent_version: Optional[str] = None,
    comparison_cluster_agent_full_image_path: Optional[str] = None,
    agent_flavor: Optional[str] = None,
    helm_config: Optional[str] = None,
    local_chart_path: Optional[str] = None,
):
    """
    Create a new EKS environment for benchmarking. It lasts around 20 minutes.

    This scenario deploys two independent Datadog Agent installations (baseline and comparison)
    in separate namespaces to enable performance comparisons. You can configure the versions
    and image paths independently for each variant using the variant-specific parameters.

    Example usage:
    - Compare two agent versions:
      inv aws.benchmarkeks --baseline-version=7.55.0 --comparison-version=7.56.0

    - Compare specific image builds:
      inv aws.benchmarkeks --baseline-full-image-path=datadog/agent:7.55.0 --comparison-full-image-path=datadog/agent:7.56.0-rc.1

    If variant-specific parameters are not provided, the scenario will fall back to the default
    agent_version and full_image_path parameters for both variants.
    """

    extra_flags = {
        "ddinfra:aws/eks/linuxARMNodeGroup": linux_arm_node_group,
        "ddinfra:aws/eks/linuxBottlerocketNodeGroup": bottlerocket_node_group,
        "ddinfra:aws/eks/linuxNodeGroup": str(linux_node_group),
        "ddinfra:aws/eks/windowsNodeGroup": windows_node_group,
        "ddagent:localChartPath": local_chart_path,
        "ddtestworkload:deployArgoRollout": install_argorollout,
        # Benchmarkeks specific parameters for baseline variant
        "ddagent:baselineVersion": baseline_version,
        "ddagent:baselineFullImagePath": baseline_full_image_path,
        "ddagent:baselineClusterAgentVersion": baseline_cluster_agent_version,
        "ddagent:baselineClusterAgentFullImagePath": baseline_cluster_agent_full_image_path,
        # Benchmarkeks specific parameters for comparison variant
        "ddagent:comparisonVersion": comparison_version,
        "ddagent:comparisonFullImagePath": comparison_full_image_path,
        "ddagent:comparisonClusterAgentVersion": comparison_cluster_agent_version,
        "ddagent:comparisonClusterAgentFullImagePath": comparison_cluster_agent_full_image_path,
    }

    # Override the instance type if specified
    # ARM node groups use defaultARMInstanceType, all others (Linux, Bottlerocket, Windows) use defaultInstanceType
    if instance_type is not None:
        if linux_arm_node_group:
            extra_flags["ddinfra:aws/defaultARMInstanceType"] = instance_type
        else:
            extra_flags["ddinfra:aws/defaultInstanceType"] = instance_type

    full_stack_name = deploy(
        ctx,
        scenario_name,
        debug=debug,
        app_key_required=True,
        stack_name=stack_name,
        install_agent=install_agent,
        install_workload=install_workload,
        agent_version=agent_version,
        extra_flags=extra_flags,
        full_image_path=full_image_path,
        cluster_agent_full_image_path=cluster_agent_full_image_path,
        agent_flavor=agent_flavor,
        helm_config=helm_config,
    )

    tool.notify(ctx, "Your benchmark EKS cluster is now created")

    _show_connection_message(ctx, full_stack_name, config_path)


def _show_connection_message(ctx: Context, full_stack_name: str, config_path: Optional[str]):
    outputs = tool.get_stack_json_outputs(ctx, full_stack_name)
    kubeconfig_output = json.loads(outputs["dd-Cluster-eks"]["kubeConfig"])
    kubeconfig_content = yaml.dump(kubeconfig_output)
    kubeconfig = f"{full_stack_name}-kubeconfig.yaml"
    f = os.open(path=kubeconfig, flags=(os.O_WRONLY | os.O_CREAT | os.O_TRUNC), mode=0o600)
    with open(f, "w") as f:
        f.write(kubeconfig_content)

    try:
        local_config = config.get_local_config(config_path)
    except ValidationError as e:
        raise Exit(f"Error in config {config.get_full_profile_path(config_path)}:{e}")

    command = f"KUBECONFIG={kubeconfig} {get_aws_wrapper(local_config.get_aws().get_account())} kubectl get nodes"

    print(f"\nYou can run the following command to connect to the benchmark EKS cluster\n\n{command}\n")

    input("Press a key to copy command to clipboard...")
    pyperclip.copy(command)


@task(help={"stack_name": doc.stack_name})
def destroy_benchmarkeks(ctx: Context, stack_name: Optional[str] = None):
    """
    Destroy a benchmark EKS environment created with invoke aws.create-benchmarkeks.
    """
    destroy(ctx, scenario_name=scenario_name, stack=stack_name)
