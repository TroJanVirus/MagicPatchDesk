# MagicPatchDesk — development handoff

## Goal

Create a fast, visually approachable *native* Windows application that browses the Xbox patch catalogue, applies JMP/IPS/EIPS/limited VCDIFF patches to XBEs and supported XISOs, and creates/converts patches. The original HTML proof of concept is **not** the target UI.

## Authoritative sources

- Application and patch engine: this repository, `core/` and `cmd/magicdesk/`.
- Public JMP corpus: https://github.com/JayYardley/Xbox-Magic-Patches-by-Jay
- Website catalogue metadata and artwork: https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php
- This project's current capabilities and limitations: `README.md` and `docs/PROJECT_STATE.md`.

## Architecture

| Area | Files | Notes |
|---|---|---|
| Windows GUI | `cmd/magicdesk/*_windows.go` | Native Win32/GDI+ UI, input/output dialogs, navigation, artwork |
| Patch formats | `core/jmp.go`, `ips.go`, `eips.go`, `xdelta.go`, `formats.go`, `convert.go` | No external runtime; VCDIFF subset only |
| XBE metadata | `core/xbe.go` | Title ID and original XBE CRC32 |
| ISO/XISO | `core/xiso.go` | XDVDFS image handling; defensive allocation limits |
| Catalog and sync | `core/catalog.go`, `core/patchhub.go` | GitHub patch sync, local caches, optional PatchHub metadata |
| Tests | `core/core_test.go`, `testdata/interop/` | Synthetic fixtures and JS/Go-format interoperability |

## Invariants

1. Never modify the user's input image or executable in place.
2. JMP record groups share a checkbox by case-insensitive comment, but individual records remain ordered; `[-]` starts disabled.
3. Verify a JMP's specified CRC32 values against the *original XBE bytes* (even when found inside an ISO).
4. Preserve local `Custom/` files during synchronization of the GitHub catalogue.
5. Keep all patch operations usable without internet; online sync and cover art are optional.
6. Make unsupported formats and unsafe ISO growth explicit errors, not silent success.
7. Keep Windows distribution portable; unsigned developer executables may trigger SmartScreen, and code signing is not solved by bundling a file.

## Known gaps

- Windows UI and console-boot compatibility have not been independently validated on real hardware.
- Xdelta3 secondary compression and custom VCDIFF instruction tables are unsupported.
- Full-disc ISO expansion is restricted; arbitrary files, compressed images, and ISO repacking are unsupported.
- No GPU patch acceleration; worker count applies to indexing and ISO copying, not sequential JMP records.
- No automated game-image regression corpus (copyright concerns); add synthetic XDVDFS and JMP tests as improvements are made.
- No executable auto-updater, presets/undo journal, batch conversions or complete game-asset editor yet.

## Recommended next actions

1. Verify launch, rendering, file dialogs, DPI and drag/drop on a clean Windows 10 and Windows 11 install.
2. Add end-to-end fixture tests from real JMP metadata (without redistribution of game bytes); test CRC32 matching and group controls.
3. Improve catalogue UX and safe/explicit ISO compatibility detection.
4. Expand image formats only with isolated parsers and round-trip tests.
5. Release signed Windows binaries via GitHub Releases when ready, rather than committing binaries to source control.

Never claim a Windows GUI run or Xbox boot passed based only on Linux unit tests or Windows cross-compilation.
