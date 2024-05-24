#!/bin/bash
# (C) Datadog, Inc. 2010-present
# All rights reserved
# Licensed under Apache-2.0 License (see LICENSE)
set -e

# Root user detection
if [ "$UID" == "0" ]; then
    sudo_cmd=""
else
    sudo_cmd="sudo"
fi
config_file="/etc/datadog-agent/datadog.yaml"
$sudo_cmd mkdir -p /etc/datadog-agent
$sudo_cmd touch $config_file
$sudo_cmd chmod 644 $config_file
$sudo_cmd sh -c "echo '${AGENT_CONFIG:-api_key: 000000000}' > $config_file" # We at least need the api_key field in the config

INSTALLER_BIN="/opt/datadog-installer/bin/installer/installer"
OCI_URL_PREFIX="oci://docker.io/datadog/"
ARCH=$(uname -m)
KNOWN_DISTRIBUTION="(Debian|Ubuntu|RedHat|CentOS|openSUSE|Amazon|Arista|SUSE|Rocky|AlmaLinux)"
DISTRIBUTION=$(lsb_release -d 2>/dev/null | grep -Eo $KNOWN_DISTRIBUTION  || grep -Eo $KNOWN_DISTRIBUTION /etc/issue 2>/dev/null || grep -Eo $KNOWN_DISTRIBUTION /etc/Eos-release 2>/dev/null || grep -m1 -Eo $KNOWN_DISTRIBUTION /etc/os-release 2>/dev/null || uname -s)
if [ -f /etc/debian_version ] || [ "$DISTRIBUTION" == "Debian" ] || [ "$DISTRIBUTION" == "Ubuntu" ]; then
    OS="Debian"
    # small hack to match datadog-agent deb testing repo, and avoid breaking e2e tests
    if [ "${ARCH}" = "aarch64" ]; then
        ARCH="arm64"
    else
        ARCH="x86_64"
    fi
elif [ -f /etc/redhat-release ] || [ "$DISTRIBUTION" == "RedHat" ] || [ "$DISTRIBUTION" == "CentOS" ] || [ "$DISTRIBUTION" == "Amazon" ] || [ "$DISTRIBUTION" == "Rocky" ] || [ "$DISTRIBUTION" == "AlmaLinux" ]; then
    OS="RedHat"
# Some newer distros like Amazon may not have a redhat-release file
elif [ -f /etc/system-release ] || [ "$DISTRIBUTION" == "Amazon" ]; then
    OS="RedHat"
# Arista is based off of Fedora14/18 but do not have /etc/redhat-release
elif [ -f /etc/Eos-release ] || [ "$DISTRIBUTION" == "Arista" ]; then
    OS="RedHat"
# openSUSE and SUSE use /etc/SuSE-release or /etc/os-release
elif [ -f /etc/SuSE-release ] || [ "$DISTRIBUTION" == "SUSE" ] || [ "$DISTRIBUTION" == "openSUSE" ]; then
    OS="SUSE"
fi

apt_url="apttesting.datad0g.com"
apt_repo_version="${DD_PIPELINE_ID}-a7-${ARCH} 7"
apt_usr_share_keyring="/usr/share/keyrings/datadog-archive-keyring.gpg"
apt_trusted_d_keyring="/etc/apt/trusted.gpg.d/datadog-archive-keyring.gpg"

MAX_RETRY_NB=10
keys_url="keys.datadoghq.com"

if [ "${OS}" = "Debian" ]; then
    $sudo_cmd DEBIAN_FRONTEND=noninteractive apt-get update
    for _ in $(seq 1 $MAX_RETRY_NB); do
        $sudo_cmd DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https curl gnupg
        apt_exit_code=$?
        if [ $apt_exit_code -ge 0 ]; then
            break
        fi
    done

    printf "\033[34m\n* Installing APT package sources for Datadog\n\033[0m\n"
    $sudo_cmd sh -c "echo 'deb [signed-by=${apt_usr_share_keyring}] https://${apt_url}/ ${apt_repo_version}' > /etc/apt/sources.list.d/datadog.list"
    $sudo_cmd sh -c "chmod a+r /etc/apt/sources.list.d/datadog.list"

    if [ ! -f $apt_usr_share_keyring ]; then
        $sudo_cmd touch $apt_usr_share_keyring
    fi
    APT_GPG_KEYS=("DATADOG_APT_KEY_CURRENT.public" "DATADOG_APT_KEY_C0962C7D.public" "DATADOG_APT_KEY_F14F620E.public" "DATADOG_APT_KEY_382E94DE.public")
    for key in "${APT_GPG_KEYS[@]}"; do
        $sudo_cmd curl --retry 5 -o "/tmp/${key}" "https://${keys_url}/${key}"
        $sudo_cmd cat "/tmp/${key}" | $sudo_cmd gpg --import --batch --no-default-keyring --keyring "$apt_usr_share_keyring"
    done

    if [ ! -f $apt_usr_share_keyring ]; then
       release_version="$(grep VERSION_ID /etc/os-release | cut -d = -f 2 | xargs echo | cut -d "." -f 1)"
    fi
    if { [ "$DISTRIBUTION" == "Debian" ] && [ "$release_version" -lt 9 ]; } || \
       { [ "$DISTRIBUTION" == "Ubuntu" ] && [ "$release_version" -lt 16 ]; }; then
        # copy with -a to preserve file permissions
        $sudo_cmd cp -a $apt_usr_share_keyring $apt_trusted_d_keyring
    fi

    $sudo_cmd DEBIAN_FRONTEND=noninteractive apt-get update
    $sudo_cmd apt-get install -y --force-yes datadog-installer

    # Only for systemd
    exit_status=0
    $sudo_cmd systemctl status datadog-installer || exit_status=$?
    if [ $exit_status -ne 4 ]; then # Status 4 means the unit does not exist
        $sudo_cmd systemctl daemon-reload
        $sudo_cmd systemctl stop datadog-installer
    fi
    # Add packages
    for pkg in ${PACKAGES[@]}; do
        $sudo_cmd $INSTALLER_BIN bootstrap --url "${OCI_URL_PREFIX}${pkg}"
    done
    if [ $exit_status -ne 4 ]; then # Status 4 means the unit does not exist
        $sudo_cmd systemctl start datadog-installer
    fi
elif [ "${OS}" = "RedHat" ]; then
    yum_url="yumtesting.datad0g.com/testing"
    yum_repo_version="${DD_PIPELINE_ID}-i7/7"

    RPM_GPG_KEYS=("DATADOG_RPM_KEY_CURRENT.public" "DATADOG_RPM_KEY_B01082D3.public" "DATADOG_RPM_KEY_FD4BF915.public" "DATADOG_RPM_KEY_E09422B3.public")
    separator='\n       '
    for key_path in "${RPM_GPG_KEYS[@]}"; do
        gpgkeys="${gpgkeys:+"${gpgkeys}${separator}"}https://${keys_url}/${key_path}"
    done
    $sudo_cmd sh -c "echo -e '[datadog]\nname = Datadog, Inc.\nbaseurl = https://${yum_url}/${yum_repo_version}/${ARCH}/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\npriority=1\ngpgkey=${gpgkeys}' > /etc/yum.repos.d/datadog.repo"
    $sudo_cmd yum -y clean metadata
    $sudo_cmd yum -y install datadog-installer
elif [ "${OS}" = "SUSE" ]; then
    yum_url="yumtesting.datad0g.com/suse/testing"
    yum_repo_version="${DD_PIPELINE_ID}-i7/7"

    RPM_GPG_KEYS=("DATADOG_RPM_KEY_CURRENT.public" "DATADOG_RPM_KEY_B01082D3.public" "DATADOG_RPM_KEY_FD4BF915.public" "DATADOG_RPM_KEY_E09422B3.public")
    separator='\n       '
    for key_path in "${RPM_GPG_KEYS[@]}"; do
        gpgkeys="${gpgkeys:+"${gpgkeys}${separator}"}https://${keys_url}/${key_path}"
    done
    $sudo_cmd sh -c "echo -e '[datadog]\nname = Datadog, Inc.\nbaseurl = https://${yum_url}/${yum_repo_version}/${ARCH}/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\npriority=1\ngpgkey=${gpgkeys}' > /etc/zypp/repos.d/datadog.repo"
    $sudo_cmd zypper -n --gpg-auto-import-keys refresh
    $sudo_cmd zypper -n install datadog-installer

    # Only for systemd
    exit_status=0
    $sudo_cmd systemctl status datadog-installer || exit_status=$?
    if [ $exit_status -ne 4 ]; then # Status 4 means the unit does not exist
        $sudo_cmd systemctl daemon-reload
        $sudo_cmd systemctl stop datadog-installer
    fi
    # Add packages
    for pkg in ${PACKAGES[@]}; do
        $sudo_cmd $INSTALLER_BIN bootstrap --url "${OCI_URL_PREFIX}${pkg}"
    done
    if [ $exit_status -ne 4 ]; then # Status 4 means the unit does not exist
        $sudo_cmd systemctl start datadog-installer
    fi
fi
