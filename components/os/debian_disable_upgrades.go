package os

import (
	_ "embed"
)

//go:embed scripts/debian-disable-unattended-upgrades.sh
var DebianDisableUnattendedUpgradesScriptContent string
