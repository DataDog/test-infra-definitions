#!/bin/bash
apt-get -y remove unattended-upgrades

# Try to disable unattended upgrades and apt automatic updates, should not fail if it is not installed
sudo systemctl disable unattented-upgrades.service || true
sudo systemctl stop unattented-upgrades.service || true

sudo systemctl disable apt-daily.service || true
sudo systemctl disable apt-daily.time || true
sudo systemctl stop apt-daily.service || true
sudo systemctl stop apt-daily.timer || true

sudo systemctl disable apt-daily-upgrade.service || true
sudo systemctl disable apt-daily-upgrade.timer || true
sudo systemctl stop apt-daily-upgrade.service || true
sudo systemctl stop apt-daily-upgrade.timer || true
