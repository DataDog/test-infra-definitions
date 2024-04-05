#!/bin/bash
# (C) Datadog, Inc. 2010-present
# All rights reserved
# Licensed under Apache-2.0 License (see LICENSE)
set -e


# Set up a named pipe for logging
npipe=/tmp/$$.tmp
mknod $npipe p

# Log all output to a log for error checking
tee <$npipe $logfile &
exec 1>&-
exec 1>$npipe 2>&1
trap "rm -f $npipe" EXIT

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
$sudo_cmd sh -c "echo 'api_key: $apikey' > $config_file"

KNOWN_DISTRIBUTION="(Debian|Ubuntu|RedHat|CentOS|openSUSE|Amazon|Arista|SUSE|Rocky|AlmaLinux)"
DISTRIBUTION=$(lsb_release -d 2>/dev/null | grep -Eo $KNOWN_DISTRIBUTION  || grep -Eo $KNOWN_DISTRIBUTION /etc/issue 2>/dev/null || grep -Eo $KNOWN_DISTRIBUTION /etc/Eos-release 2>/dev/null || grep -m1 -Eo $KNOWN_DISTRIBUTION /etc/os-release 2>/dev/null || uname -s)
if [ -f /etc/debian_version ] || [ "$DISTRIBUTION" == "Debian" ] || [ "$DISTRIBUTION" == "Ubuntu" ]; then
    OS="Debian"
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

ARCH=$(uname -m)
if [ "${ARCH}" = "aarch64" ]; then
    ARCH="arm64"
fi

apt_url="apttesting.datad0g.com"
apt_repo_version="${DD_PIPELINE_ID}-u7-${ARCH} 7"
apt_usr_share_keyring="/usr/share/keyrings/datadog-archive-keyring.gpg"
apt_trusted_d_keyring="/etc/apt/trusted.gpg.d/datadog-archive-keyring.gpg"

yum_url="yumtesting.datad0g.com/testing"
yum_repo_version="${DD_PIPELINE_ID}-u7/7"

DD_APT_INSTALL_ERROR_MSG=/tmp/ddog_install_error_msg
MAX_RETRY_NB=10
keys_url="keys.datadoghq.com"

if [ "${OS}" = "Debian" ]; then
    printf "\033[34m\n* Installing APT package sources for Datadog\n\033[0m\n"
    $sudo_cmd sh -c "echo 'deb [signed-by=${apt_usr_share_keyring}] https://${apt_url}/ ${apt_repo_version}' > /etc/apt/sources.list.d/datadog.list"
    $sudo_cmd sh -c "chmod a+r /etc/apt/sources.list.d/datadog.list"

    if [ ! -f $apt_usr_share_keyring ]; then
        $sudo_cmd touch $apt_usr_share_keyring
    fi
    # ensure that the _apt user used on Ubuntu/Debian systems to read GPG keyrings
    # can read our keyring
    $sudo_cmd chmod a+r $apt_usr_share_keyring

    APT_GPG_KEYS=("DATADOG_APT_KEY_CURRENT.public" "DATADOG_APT_KEY_C0962C7D.public" "DATADOG_APT_KEY_F14F620E.public" "DATADOG_APT_KEY_382E94DE.public")
    for key in "${APT_GPG_KEYS[@]}"; do
        $sudo_cmd curl --retry 5 -o "/tmp/${key}" "https://${keys_url}/${key}"
        $sudo_cmd cat "/tmp/${key}" | $sudo_cmd gpg --import --batch --no-default-keyring --keyring "$apt_usr_share_keyring"
    done
    release_version="$(grep VERSION_ID /etc/os-release | cut -d = -f 2 | xargs echo | cut -d "." -f 1)"
    if { [ "$DISTRIBUTION" == "Debian" ] && [ "$release_version" -lt 9 ]; } || \
       { [ "$DISTRIBUTION" == "Ubuntu" ] && [ "$release_version" -lt 16 ]; }; then
        # copy with -a to preserve file permissions
        $sudo_cmd cp -a $apt_usr_share_keyring $apt_trusted_d_keyring
    fi

    for i in $(seq 1 $MAX_RETRY_NB); do
        printf "\033[34m\n* Installing apt-transport-https, curl and gnupg\n\033[0m\n"
        $sudo_cmd apt-get update || printf "\033[31m\"apt-get update\" failed, the script will not install the latest version of apt-transport-https.\033[0m\n"
        apt_exit_code=0
        if [ -z "$sudo_cmd" ]; then
            DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https curl gnupg 2>$DD_APT_INSTALL_ERROR_MSG  || apt_exit_code=$?
        else
            $sudo_cmd DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https curl gnupg 2>$DD_APT_INSTALL_ERROR_MSG || apt_exit_code=$?
        fi

        if grep "Could not get lock" $DD_APT_INSTALL_ERROR_MSG; then
            RETRY_TIME=$((i*5))
            printf "\033[31mInstallation failed: Unable to get lock.\nRetrying in ${RETRY_TIME}s ($i/$MAX_RETRY_NB).\033[0m\n"
            sleep $RETRY_TIME
        elif [ $apt_exit_code -ne 0 ]; then
            cat $DD_APT_INSTALL_ERROR_MSG
            exit $apt_exit_code
        else
            break
        fi
    done
    $sudo_cmd apt-get install -y --force-yes "datadog-updater" 2> >(tee /tmp/ddog_install_error_msg >&2)
elif [ "${OS}" = "RedHat" ]; then
    RPM_GPG_KEYS=("DATADOG_RPM_KEY_CURRENT.public" "DATADOG_RPM_KEY_B01082D3.public" "DATADOG_RPM_KEY_FD4BF915.public" "DATADOG_RPM_KEY_E09422B3.public")
    separator='\n       '
    for key_path in "${RPM_GPG_KEYS[@]}"; do
        gpgkeys="${gpgkeys:+"${gpgkeys}${separator}"}https://${keys_url}/${key_path}"
    done
    $sudo_cmd sh -c "echo -e '[datadog]\nname = Datadog, Inc.\nbaseurl = https://${yum_url}/${yum_repo_version}/${ARCH}/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\npriority=1\ngpgkey=${gpgkeys}' > /etc/yum.repos.d/datadog.repo"
    $sudo_cmd yum -y clean metadata
    $sudo_cmd yum -y install datadog-updater
fi

