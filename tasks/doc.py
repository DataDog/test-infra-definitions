from . import tool

install_agent: str = f"Install the Agent (default {tool.get_default_agent_install()})."
install_workload: str = f"Install test workload (default {tool.get_default_workload_install()})."
pipeline_id: str = (
    "The pipeline id of the custom Agent build for example '16497585' (may be taken form the gitlab url)'"
)
agent_version: str = "The version of the Agent for example '7.42.0~rc.1-1' or '6.39.0 (default `latest`)'"
container_agent_version: str = "The container version of the Agent for example '7.45.0-rc.3' (default `latest`)'"
stack_name: str = "An optional name for the stack. This parameter is useful when you need to create several environments. Note: 'invoke destroy' may not work properly"
debug: str = "Launch pulumi with debug mode. Default False"
stack_name: str = (
    "An optional name for the stack. This parameter is useful when you need to create several environments."
)
os_family: str = (
    f"The operating system. Possible values are {tool.get_os_families()}. Default '{tool.get_default_os_family()}'"
)
linux_node_group: str = "Install a Linux node group (default True)"
linux_arm_node_group: str = "Install a Linux ARM node group (default False)"
bottlerocket_node_group: str = "Install a bottlerocket node group (default True)"
windows_node_group: str = "Install a Windows node group (default False)"
use_fargate: str = "Use Fargate (default True)"
fakeintake: str = "Use a dedicated fake Datadog intake (default False)"
ami_id: str = "A full Amazon Machine Image (AMI) id (e.g. ami-0123456789abcdef0)"
architecture: str = f"The architecture to use. Possible values are {tool.get_architectures()}. Default '{tool.get_default_architecture()}'"
yes: str = "Automatically approve and perform the destroy without previewing it"
use_aws_vault: str = "Wrap aws command with aws-vault, default to True"
interactive: str = "Enable interactive mode, if set to False notifications and copy to clipboard are disabled"
config_path: str = "Specify a custom config path to use"
use_loadBalancer: str = "Use a loadBalancer to instantiate the fakeintake (default False)"
