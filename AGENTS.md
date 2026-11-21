# MagicPatchDesk: instructions for coding agents

Read `README.md`, `AI_HANDOFF.md`, and `docs/PROJECT_STATE.md` before making changes. This repository is the authoritative source of the *native Windows* Magic Patch Desk app, not the earlier HTML workbench.

## Source and build

- Language: Go 1.23+; use only standard-library dependencies unless a change requires another library and its distribution implications are documented.
- Native GUI: `cmd/magicdesk` (Windows Win32/GDI/GDI+; Windows-only source files).
- Patch logic: `core/`; make it OS-independent and covered by Go tests.
- Format interoperability checks: `cmd/interop/` and `testdata/interop/`.
- Run `go test ./...` after every core change; optionally `go test -race ./core` on a supported host.
- Windows build: `BUILD-WINDOWS.cmd` on Windows, or `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -s -w' -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk` on other hosts.
- Do not commit generated executables, ISO images, XBEs, private game assets, downloaded patch caches, or local user files. Put public binaries in GitHub Releases.
- Do not add automatic GitHub Actions workflows unless requested. If CI is added, prefer manual or carefully scoped runs to conserve Actions minutes.

## Compatibility contracts

- JMP: preserve `system`, `game-title`, `region`, `version`, `author`, and `notes`; the optional `{CRC32}` values in `version`; search-and-replace (first occurrence), hexadecimal offsets, APPEND, and `[-]` default-disabled records.
- Identically named JMP comments must appear as one group toggle (case-insensitive), while records execute in their original sequence.
- Validate XBE CRC32 and Title ID where specified. Never silently override mismatches.
- Never overwrite the selected original XBE or ISO; stage writes, report errors, and produce an independent patched output.
- Handle XISO/XDVDFS layouts defensively. Do not falsely claim support for compressed CCI/CHD or every xdelta3 stream.
- Keep offline use functional without GitHub or PatchHub; local JMPs in `%APPDATA%\\MagicPatchDesk\\Custom` must work even if sync fails.
- Preserve original patch author credits and links. Do not bundle copyrighted Xbox game images or executables.

## Testing and reporting

For bug reports, record the game title, Title ID, original XBE CRC32, JMP filename, selected groups, and whether the input was `.xbe`, game-partition XISO, or full-disc ISO. Document any unverified hardware, Windows GUI, or game-boot claims explicitly in `docs/PROJECT_STATE.md`.


## Multi-file JMP rules

A comment naming `settings.cfg` targets that file. Prompt the user to locate/use, skip, or cancel; never silently apply that record to the default XBE. Reject ambiguous/unsafe paths and preserve the original. In an XISO resolve by unique relative path or basename. Multiple file targets must not be collapsed into one IPS, BPS or Xdelta patch without separating the records.


## v0.6 file-target packs and GUI feedback

Read `docs/PATCH_PACK_FORMAT.md`. A JMP is always one file, even if its records target multiple distinct files. For non-JMP exports containing named targets, create a `.mpdpack` with one standard patch per target and a manifest; never force different files into one IPS/BPS/Xdelta stream. Inspect `notes=` and each record comment for filename.ext references. Per-record wins, one global note target is inherited by otherwise unnamed records, ambiguous notes fail closed. Do not package game binaries or silently skip file targets. Preserve source requirements, SHA256 integrity, and optional JMP CRC32 constraints. Native GDI controls now respond to mouse hover/down/up: do not break pressed-state click dispatch or 16 ms animation timer. Test with `go test ./...`, `go test -race ./core`, `go vet ./...` and cross-build Windows x64.
