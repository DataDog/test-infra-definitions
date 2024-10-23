package os

import (
	_ "embed"
)

//go:embed scripts/apt-disable-unattended-upgrades.sh
var APTDisableUnattendedUpgradesScriptContent string
