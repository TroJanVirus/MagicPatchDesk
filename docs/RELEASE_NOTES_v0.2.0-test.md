# v0.2.0 experimental branch — preview, records, preferences

This is a developer test branch, **not** the stable v0.1.0 release.

## New

- **Patch preview:** new PREVIEW button performs a memory-only dry run on a selected XBE or embedded ISO `default.xbe`. Shows source and expected output CRC32, sizes, modified/appended bytes, and up to eight selected JMP record positions. Missing search patterns and CRC/Title-ID mismatches cause an error *before* output is written.
- **Apply confirmation:** clicking APPLY PATCH runs preflight and asks for confirmation before writing the patched XBE or XISO. Unsafe growth within full-disc ISO is detected in preflight.
- **Per-record editing:** click + on a grouped JMP comment to expand its individual entries; each record can be toggled without losing its group association. Mixed on/off states are shown.
- **Saved settings:** worker count and catalogue filter persist in `%APPDATA%\MagicPatchDesk\settings.json`. CRC/Title-ID override does not persist.

## Testing

- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -s -w' -o MagicPatchDesk-Windows-v0.2.0-test.exe ./cmd/magicdesk`

No graphical Windows runtime or original Xbox hardware was available for manual QA. Please test the interface and any ISO image behaviour with copies of your own game files.
