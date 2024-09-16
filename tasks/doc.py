from . import tool

install_agent: str = f"Install the Agent (default {tool.get_default_agent_install()})."
install_agent_with_operator: str = (
    f"Install the Agent with Operator (default {tool.get_default_agent_with_operator_install()})."
)
install_installer: str = "Install the Installer (default False)."
install_workload: str = f"Install test workload (default {tool.get_default_workload_install()})."
pipeline_id: str = (
    "The pipeline id of the custom Agent build for example '16497585' (may be taken form the gitlab url)'"
)
job_name: str = "Name of the job within the agent pipeline for example 'deploy_deb_testing-a7_x64'"
agent_version: str = "The version of the Agent for example '7.42.0~rc.1-1' or '6.39.0 (default `latest`)'"
container_agent_version: str = "The container version of the Agent for example '7.45.0-rc.3' (default `latest`)'"
stack_name: str = "An optional name for the stack. This parameter is useful when you need to create several environments. Note: 'invoke destroy' may not work properly"
debug: str = "Launch pulumi with debug mode. Default False"
linux_node_group: str = "Install a Linux node group (default True)"
linux_arm_node_group: str = "Install a Linux ARM node group (default False)"
bottlerocket_node_group: str = "Install a bottlerocket node group (default True)"
windows_node_group: str = "Install a Windows node group (default False)"
fakeintake: str = "Use a dedicated fake Datadog intake (default False)"
yes: str = "Automatically approve and perform the destroy without previewing it"
use_aws_vault: str = "Wrap aws command with aws-vault, default to True"
interactive: str = "Enable interactive mode, if set to False notifications and copy to clipboard are disabled"
config_path: str = "Specify a custom config path to use"
use_loadBalancer: str = "Use a loadBalancer to instantiate the fakeintake (default False)"
clean_known_hosts: str = "Clean the host from ssh known_hosts file after destroying the VM (default True)"
no_verify: str = "Do not verify deploy jobs before creating vm"
debug: str = "Check for common errors in your environment setup and configuration (defualt False)"
site: str = "Datadog site to contact (default 'datad0g.com')"
ssh_user: str = "The user to use for ssh connection (default will be selected depending on the OS family). Should only be used if you explicitly need to use a different user"
os_version: str = "The version of the OS to use (default will be selected depending on the OS family). See https://github.com/DataDog/test-infra-definitions/blob/main/components/os/linux_descriptors.go for a list of version available for a given OS (https://github.com/DataDog/test-infra-definitions/blob/main/components/os/windows_descriptors.go for Windows)"
full_image_path: str = "The full image path (registry:tag) of the Agent image to deploy"
cluster_agent_full_image_path: str = "The full image path (registry:tag) of the Cluster Agent image to deploy"
