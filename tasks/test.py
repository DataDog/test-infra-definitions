from invoke.tasks import task
from invoke.exceptions import Exit
import difflib

try:
    from termcolor import colored
except ImportError:
    colored = lambda *args: args[0]


@task(
    iterable="replacements",
    help={
        "xslt": "XSLT file to use for transformation",
        "replacements": "Override replacements to use for transformation. Multiple arguments of the form key=value",
    },
)
def check_xslt(ctx, xslt, replacements=None):
    """
    Checks the XSLT transformations in the scenarios/aws/microVMs/microvms/resources path
    Useful for testing and checking transformation there without running the full pulumi stack
    """
    base_xml = """
<domain type="kvm">
    <name>local-ddvm-local-ubuntu_22.04-distro_local-ddvm-4-8192</name>
    <memory unit="MiB">8192</memory>
    <vcpu>4</vcpu>
    <os firmware="efi">
        <type machine="virt">hvm</type>
    </os>
    <features>
        <pae></pae>
        <acpi></acpi>
        <apic></apic>
    </features>
    <cpu></cpu>
    <devices>
        <disk type="volume" device="disk">
            <driver name="qemu" type="qcow2"></driver>
            <source pool="local-ddvm-global-pool" volume="local-ddvm-global-pool-ubuntu_22.04-distro_local-local-final-overlay-ubuntu_22.04-05d642c"></source>
            <target dev="vda" bus="virtio"></target>
        </disk>
        <disk type="volume" device="disk">
            <driver name="qemu" type="qcow2"></driver>
            <source pool="local-ddvm-global-pool" volume="local-ddvm-global-pool-docker-arm64.qcow2-distro_local-local-final-overlay-ubuntu_22.04-9c703cf"></source>
            <target dev="vdb" bus="virtio"></target>
        </disk>
        <console>
            <target type="serial" port="0"></target>
        </console>
        <channel type="unix">
            <target type="virtio" name="org.qemu.guest_agent.0"></target>
        </channel>
        <rng model="virtio">
            <backend model="random">/dev/urandom</backend>
        </rng>
    </devices>
</domain>
"""

    try:
        import lxml.etree as etree
    except ImportError:
        raise Exit("lxml is not installed. Please install it with `pip install lxml`")

    parser = etree.XMLParser(remove_blank_text=True)
    dom = etree.fromstring(base_xml, parser)

    default_replacements = {
        "sharedFSMount": "/opt/kernel-version-testing",
        "domainID": "local-ddvm-local-ubuntu_22.04-distro_local-ddvm-4-8192",
        "mac": "52:54:00:00:00:00",
        "nvram": "/tmp/nvram",
        "efi": "/tmp/efi",
        "vcpu": "4",
        "cputune": "<cputune></cputune>",
        "hypervisor": "hvf",
    }

    for repl in replacements or []:
        key, value = repl.split("=")
        default_replacements[key] = value

    with open(xslt, "r") as f:
        data = f.read()

    for key, value in default_replacements.items():
        data = data.replace("{%s}" % key, value)

    xslt = etree.fromstring(data, parser)
    transform = etree.XSLT(xslt)
    newdom = transform(dom)

    orig_xml = etree.tostring(dom, pretty_print=True).decode('utf-8').replace('\\n', '\n')
    new_xml = etree.tostring(newdom, pretty_print=True).decode('utf-8').replace('\\n', '\n')

    print(colored("=== Original XML ===", "white"))
    print(orig_xml)
    print(colored("=== Transformed XML ===", "white"))
    print(new_xml)

    diff = difflib.unified_diff(orig_xml.split('\n'), new_xml.split('\n'), fromfile="original", tofile="transformed")

    print(colored("=== Diff ===", "white"))

    for line in diff:
        line = line.rstrip('\n')

        if line.startswith('-'):
            print(colored(line, "red"))
        elif line.startswith('+'):
            print(colored(line, "green"))
        elif line.startswith('@@'):
            print(colored(line, "blue"))
        else:
            print(line)
