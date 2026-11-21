#!/usr/bin/env bash
set -euo pipefail
[ -f core/update.go ] && [ -f cmd/magicdesk/maintenance_windows.go ]
grep -q 'const StableReleaseAsset = "MagicPatchDesk-Windows-x64.exe"' core/update.go
grep -q 'const UpdateProtocol = 1' core/update.go
grep -q -- '--mpd-apply-update-v1' cmd/magicdesk/maintenance_windows.go
grep -q -- '--mpd-healthcheck-v1' cmd/magicdesk/maintenance_windows.go
grep -q 'func main()' cmd/magicdesk/main_windows.go
grep -q 'helperMode()' cmd/magicdesk/main_windows.go
printf 'Self-update CLI and release artifact compatibility contract present.\n'
