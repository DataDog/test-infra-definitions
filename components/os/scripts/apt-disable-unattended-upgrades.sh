#!/bin/bash

# Try to disable unattended upgrades and apt automatic updates, should not fail if it is not installed
sudo systemctl disable unattended-upgrades.service || true
sudo systemctl stop unattended-upgrades.service || true

sudo systemctl disable apt-daily.service || true
sudo systemctl disable apt-daily.time || true
sudo systemctl stop apt-daily.service || true
sudo systemctl stop apt-daily.timer || true

sudo systemctl disable apt-daily-upgrade.service || true
sudo systemctl disable apt-daily-upgrade.timer || true
sudo systemctl stop apt-daily-upgrade.service || true
sudo systemctl stop apt-daily-upgrade.timer || true

# Send the TERM signal to any remaining unattended-upgrades processes
pgrep unattended-upgrades | xargs -r -n 1 -t kill -TERM || true

max_to_wait=10
while pgrep unattended-upgrades && [ $max_to_wait -gt 0 ]; do
	echo "Waiting for unattended-upgrades to terminate"
	sleep 1
	max_to_wait=$((max_to_wait - 1))
done

# Kill any unattended-upgrades processes that didn't terminate
pgrep unattended-upgrades | xargs -r -n 1 -t kill -KILL || true

apt-get -y purge unattended-upgrades || true

# Ensure the lock files are removed
rm -f /var/lib/apt/lists/lock || true
rm -f /var/cache/apt/archives/lock || true
rm -f /var/lib/dpkg/lock || true
rm -f /var/lib/dpkg/lock-frontend || true
rm -f /var/cache/apt/archives/partial/lock || true
