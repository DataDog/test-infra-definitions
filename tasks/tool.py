import getpass
import json
import os
import pathlib
import platform
from io import StringIO
from typing import Any, List, Optional, Union

import pyperclip
from invoke.context import Context
from invoke.exceptions import Exit
from termcolor import colored


def is_windows():
    return platform.system() == "Windows"


if is_windows():
    try:
        # Explicitly enable terminal colors work on Windows
        # os.system() seems to implicitly enable them, but ctx.run() does not
        from colorama import just_fix_windows_console

        just_fix_windows_console()
    except ImportError:
        print(
            "colorama is not up to date, terminal colors may not work properly. Please run 'pip install -r requirements.txt' to fix this."
        )


def ask(question: str) -> str:
    return input(colored(question, "blue"))


def debug(msg: str):
    print(colored(msg, "white"))


def info(msg: str):
    print(colored(msg, "green"))


def warn(msg: str):
    print(colored(msg, "yellow"))


def error(msg: str):
    print(colored(msg, "red"))


def get_os_families() -> List[str]:
    return [
        get_default_os_family(),
        "windows",
        "amazonlinux",
        "amazonlinuxdocker",
        "debian",
        "redhat",
        "suse",
        "fedora",
        "centos",
        "rockylinux",
    ]


def get_package_for_os(os: str) -> str:
    package_map = {
        get_default_os_family(): "deb",
        "windows": "windows",
        "amazonlinux": "rpm",
        "amazonlinuxdocker": "rpm",
        "debian": "deb",
        "redhat": "rpm",
        "suse": "suse_rpm",
        "fedora": "rpm",
        "centos": "rpm",
        "rockylinux": "rpm",
    }

    return package_map[os]


def get_deploy_job(os: str, arch: Union[str, None], agent_version: Union[str, None] = None) -> str:
    """
    Returns the deploy job name within the datadog agent repo that creates
    images used in create-vm
    """
    pkg = get_package_for_os(os)
    if agent_version is None:
        v = 'a7'
    else:
        major = agent_version.split('.')[0]
        assert major in ('6', '7'), f'Invalid agent version {agent_version}'
        v = f'a{major}'

    if arch == 'x86_64':
        arch = 'x64'

    # Construct job name
    if os == 'windows':
        suffix = f'-{v}'
        assert arch == 'x64', f'Invalid architecure {arch} for Windows'
    elif os == 'suse':
        suffix = f'_{arch}-{v}'
    elif pkg in ('deb', 'rpm'):
        suffix = f'-{v}_{arch}'
    else:
        raise RuntimeError(f'Cannot deduce deploy job from {os}::{arch}')

    return f'deploy_{pkg}_testing{suffix}'


def get_default_os_family() -> str:
    return "ubuntu"


def get_architectures() -> List[str]:
    return [get_default_architecture(), "arm64"]


def get_default_architecture() -> str:
    return "x86_64"


def get_default_agent_install() -> bool:
    return True


def get_default_workload_install() -> bool:
    return True


def get_stack_name(stack_name: Optional[str], scenario_name: str) -> str:
    if stack_name is None:
        stack_name = scenario_name.replace("/", "-")
    # The scenario name cannot start with the stack name because ECS
    # stack name cannot start with 'ecs' or 'aws'
    return f"{get_stack_name_prefix()}{stack_name}"


def get_stack_name_prefix() -> str:
    user_name = f"{getpass.getuser()}-"
    return user_name.replace(".", "-")  # EKS doesn't support '.'


def get_stack_json_outputs(ctx: Context, full_stack_name: str) -> Any:
    buffer = StringIO()

    cmd_parts: List[str] = ["pulumi", "stack", "output", "--json", "-s", full_stack_name, get_pulumi_dir_flag()]
    ctx.run(
        " ".join(cmd_parts),
        out_stream=buffer,
    )
    return json.loads(buffer.getvalue())


def get_stack_json_resources(ctx: Context, full_stack_name: str) -> Any:
    buffer = StringIO()
    with ctx.cd(_get_root_path()):
        ctx.run(
            f"pulumi stack export -s {full_stack_name}",
            out_stream=buffer,
        )
    out = json.loads(buffer.getvalue())
    return out['deployment']['resources']


def get_aws_wrapper(
    aws_account: str,
) -> str:
    return f"aws-vault exec sso-{aws_account}-account-admin -- "


def is_linux():
    return platform.system() == "Linux"


def is_wsl():
    return "microsoft" in platform.uname().release.lower()


def get_aws_instance_password_data(
    ctx: Context, vm_id: str, key_path: str, aws_account: Optional[str] = None, use_aws_vault: Optional[bool] = True
) -> str:
    buffer = StringIO()
    with ctx.cd(_get_root_path()):
        cmd = f"aws ec2 get-password-data --instance-id {vm_id} --priv-launch-key {key_path}"
        if use_aws_vault:
            if aws_account is None:
                raise Exit("AWS account is required when using aws-vault.")
            cmd = get_aws_wrapper(aws_account) + cmd
        ctx.run(cmd, out_stream=buffer)
    out = json.loads(buffer.getvalue())
    return out["PasswordData"]


def get_image_description(ctx: Context, ami_id: str) -> Any:
    buffer = StringIO()
    ctx.run(
        f"aws-vault exec sso-agent-sandbox-account-admin -- aws ec2 describe-images --image-ids {ami_id}",
        out_stream=buffer,
    )
    result = json.loads(buffer.getvalue())
    if len(result["Images"]) > 1:
        raise Exit(f"The AMI id {ami_id} returns more than one definition.")
    else:
        return result["Images"][0]


def rdp(ctx, ip):
    if is_windows() or is_wsl():
        rdp_windows(ctx, ip)
    elif is_linux():
        raise Exit("RDP is not yet implemented on Linux")
    else:
        rdp_macos(ctx, ip)


def rdp_windows(ctx, ip):
    ctx.run(f"mstsc.exe /v:{ip}", disown=True)


def rdp_macos(ctx, ip):
    ctx.run(f"open -a '/Applications/Microsoft Remote Desktop.app' rdp://{ip}", disown=True)


def notify(ctx, text):
    if is_linux():
        notify_linux(ctx, text)
    elif is_windows():
        notify_windows()
    else:
        notify_macos(ctx, text)


def notify_macos(ctx, text):
    CMD = '''
    on run argv
    display notification (item 2 of argv) with title (item 1 of argv)
    end run
    '''
    ctx.run(f"osascript -e '{CMD}' test-infra-definitions '{text}'")


def notify_linux(ctx, text):
    ctx.run(f"notify-send 'test-infra-definitions' '{text}'")


def notify_windows():
    # TODO: Implenent notification on windows. Would require windows computer (with desktop) to test
    return


# ensure we run pulumi from a directory with a Pulumi.yaml file
# defaults to the project root directory
def get_pulumi_dir_flag():
    root_path = _get_root_path()
    current_path = os.getcwd()
    if not os.path.isfile(os.path.join(current_path, "Pulumi.yaml")):
        return f"-C {root_path}"
    return ""


def _get_root_path() -> str:
    folder = pathlib.Path(__file__).parent.resolve()
    return str(folder.parent)


class RemoteHost:
    def __init__(self, name, stack_outputs: Any):
        remoteHost: Any = stack_outputs[f"dd-Host-{name}"]
        self.host: str = remoteHost["address"]
        self.user: str = remoteHost["username"]


def show_connection_message(
    ctx: Context, remote_host_name: str, full_stack_name: str, copy_to_clipboard: Optional[bool] = True
):
    outputs = get_stack_json_outputs(ctx, full_stack_name)
    remoteHost = RemoteHost(remote_host_name, outputs)
    host = remoteHost.host
    user = remoteHost.user

    command = f"ssh {user}@{host}"

    print(f"\nYou can run the following command to connect to the host `{command}`.\n")
    if copy_to_clipboard:
        input("Press a key to copy command to clipboard...")
        pyperclip.copy(command)


def clean_known_hosts(host: str) -> None:
    """
    Remove the host from the known_hosts file.
    """
    home = os.environ.get("HOME", f"/Users/{getpass.getuser()}")
    with open(f"{home}/.ssh/known_hosts") as f:
        lines = f.readlines()

    filtered_lines = [line for line in lines if not line.startswith(host)]
    with open(f"{home}/.ssh/known_hosts", "w") as f:
        f.writelines(filtered_lines)


def get_host(ctx: Context, remote_host_name: str, scenario_name: str, stack_name: Optional[str] = None) -> str:
    """
    Get the host of the VM.
    """
    full_stack_name = get_stack_name(stack_name, scenario_name)
    outputs = get_stack_json_outputs(ctx, full_stack_name)
    remoteHost = RemoteHost(remote_host_name, outputs)
    return remoteHost.host
