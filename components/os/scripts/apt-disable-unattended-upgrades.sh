#!/bin/bash
apt-get -y remove unattended-upgrades
sudo systemctl disable unattented-upgrades.service
sudo systemctl stop unattented-upgrades.service

sudo systemctl disable apt-daily.service
sudo systemctl disable apt-daily.timer
sudo systemctl stop apt-daily.service
sudo systemctl stop apt-daily.timer

sudo systemctl disable apt-daily-upgrade.service
sudo systemctl disable apt-daily-upgrade.timer
sudo systemctl stop apt-daily-upgrade.service
sudo systemctl stop apt-daily-upgrade.timer
