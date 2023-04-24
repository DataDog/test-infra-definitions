from . import tool

install_agent: str = f"Install the Agent (default {tool.get_default_agent_install()})."
agent_version: str = "The version of the Agent for example '7.42.0~rc.1-1' or '6.39.0 (default `latest`)'"
stack_name: str = "An optional name for the stack. This parameter is useful when you need to create several environments. Note: 'invoke destroy' may not work properly"
os_type: str = f"The operating system. Possible values are {tool.get_os_types()}. Default '{tool.get_default_os_type()}'"
debug: str = "Launch pulumi with debug mode. Default False"
