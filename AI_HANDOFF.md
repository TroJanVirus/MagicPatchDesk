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

## v0.5 developer handoff — 2026-11-21

New `core/targets.go` parses filename-with-extension record comments into target names, enforces path safety and offers `Patch.ApplyToTarget`/`PreviewTarget` for named files. Single-file `Apply`/`Preview` must reject enabled named targets via `CheckSingleFileOnly`.

`cmd/magicdesk/targeted_windows.go` intercepts multi-target JMP operations before the regular XBE preview/apply flow, prompts Yes=locate/use ISO member, No=skip target, Cancel=abort, then builds separate output copies after per-file preflight. `core/xiso_multi.go` updates multiple existing XDVDFS directory entries in an output image, allowing grow/relocate only in a game-partition XISO. The UI blocks source-required conversion, warns about conversion limitations, and does not allow a multi-target JMP to be collapsed into one IPS. Test synthetic fixtures first; still needs physical Windows testing, xemu and hardware game validation. Avoid distributing copyrighted game files.

## 2026-11-21 v0.6: multi-target patch packs / filename notes / reactive Win32 UI

- `core/targets.go`: `FileTargetsInText`, `Patch.RecordTarget`. Every comment and `notes=` is scanned for filename.extension references. Ambiguous notes require explicit per-record target. URL links ignored; unsafe paths fail.
- `core/pack.go`: `PlanJMPPack`, `BuildJMPPatchPack`, `ReadPatchPack`, `ApplyPatchPack`. `.mpdpack` is ZIP + manifest + per-target IPS/BPS/Xdelta/EIPS; optional source SHA256/result hash; default-XBE JMP CRC32 retained in manifest. No original game data in archive. Full specification: `docs/PATCH_PACK_FORMAT.md`.
- `cmd/magicdesk/pack_windows.go`: source-aware per-target pack conversion, per-target file selection; disabled Convert until required originals available.
- `cmd/magicdesk/pack_apply_windows.go`: explicit full-pack application via existing XDVDFS patching / separate external copy pipeline; refuses missing targets.
- `cmd/magicdesk/main_windows.go` and `paint_windows.go`: custom eased mouse hover, clicked-depth states, real hand cursor and responsive game/chip/sidebar highlights. Do not conflate Win32 GDI UI with CSS; no browser used.
- Build and run core tests on Linux; manual Windows 10/11 and xemu/hardware validation outstanding.

## 2026-11-21 — v0.7 Rocky5 PF and all-variants pack

See docs/PF_FORMAT.md and docs/PATCH_PACK_V2.md. The PF parser/interpreter is distinct from JMP, and includes CP/HR/OF/SM/SR/MV/RN/RM. Its full-folder application stages a separate tree; ISO file operation changes are refused. New pack schema magicpatchdesk-pack-v2 includes original JMP plus all representable IPS/BPS/Xdelta/EIPS alternatives per named target and source/result checksums; v1 remains supported. Test go test ./..., race, vet and cross-build before releases.

## v0.8: local diagnostic reports

`core/diagnostics.go` classifies failures into a stable privacy-restricted `magicpatchdesk-diagnostic-v1` JSON report; `cmd/magicdesk/diagnostics_windows.go` records errors from all supported patch paths and PF warning counts. Reports are local at `<appdata>/MagicPatchDesk/Reports`, created with a unique random ID; Settings offers view/copy/open/report. `Report issue` opens an explicitly reviewed, prefilled GitHub issue and never posts automatically. Tests check private string exclusion, duplicate creation and warning metadata. Do not add automatic telemetry without explicit approval.


## v0.9: local diagnostic grouping / GitHub duplicate discovery

`core/diagnostic_groups.go` makes stable `MPD-...` group IDs from class+operation+Title ID and groups up to 1000 recent local event reports without rewriting older v0.8 records. `core/diagnostics.go` still stores one privacy-redacted JSON per event. `cmd/magicdesk/diagnostics_windows.go` and Settings expose group navigation, count, safe summaries, and opt-in GitHub issue search / new issue. The issue title contains the stable ID so later reports are searchable together. No issue/Actions automation or telemetry is introduced. Do not describe the grouping ID as anonymized; anyone who knows a Title ID and classification can reproduce it. Windows smoke tests remain outstanding.


## v0.10 — developer dashboard

`core/developer_dashboard.go` parses *public* diagnostic GitHub issues as a
read-only, explicitly refreshed snapshot, combines local grouped reports,
links related groups through `Root cause: #N` / `Related groups:` issue-body
markers, tracks a referenced `Fix PR: #N`, and sorts cases without mixing local
occurrences with public issue counts. The UI `cmd/magicdesk/developer_dashboard_windows.go`
adds the fifth workspace tab (**Diagnostics**) and prepares manual browser
workflows rather than making privileged changes. No cached auth or automatic
issue posting. Run Go tests, race/vet and Windows build; Windows GUI smoke test
is outstanding. Full user workflow: docs/DEVELOPER_DASHBOARD.md.

## 2026-11-21 v0.11 — native Recent Files QoL

Added local MRU history (`core/recent.go`, `core/recent_test.go`) with max 18 uniquely addressed files, atomic JSON persistence, optional clearing. The native sidebar Recent workspace in `cmd/magicdesk/recent_windows.go` reloads original/patch/conversion inputs, opens output folders, allows forgetting/clearing without deleting files, and handles Ctrl+O/P/F/R, Ctrl+1–6, F5 and Esc. Existing validation requirements and source/patch selection remain unchanged; paths are stored only on the local machine, never in diagnostic reports. Build with `go test ./...`, `go test -race ./core`, `go vet ./...`, Windows cross-build. Physical Windows UI verification pending.


## v0.12: Self-update/CCI/auto-report handoff

`core/update.go` holds the mandatory permanent asset/version/checksum contract; `cmd/magicdesk/maintenance_windows.go` handles installation, check/update, a separate app-copy helper CLI and rollback. Preserve these in every subsequent release. `cmd/magicdesk/maintenance_paint_windows.go` is the native Maintenance UI. `cmd/magicdesk/report_auth_windows.go` stores an opt-in GitHub Issues token in Windows Credential Manager and `core/report_sync.go` creates/comments on group-specific issues using only privacy-redacted reports. `core/cci.go` implements CCIM v1 read/write/extract with defensive sector/index validation; `cmd/magicdesk/cci_patch_windows.go` handles direct default-XBE CCI patching. Read docs/UPDATE_CONTRACT.md, docs/CCI_SUPPORT.md, docs/AUTO_REPORTS.md. Preserve all file format safety guards and never require a Github token for offline patching. Install folder and roaming user data are completely separate. Real Win10, Cerbios hardware and authenticated GitHub posting still need manual testing.

## Planned — independent game output formats (2026-11-22)

See `docs/IMAGE_OUTPUT_WORKFLOW.md` and GitHub issue #9. An XISO/CCI/ISO/folder/XBE input must not lock the *game* output format. Let users explicitly select patched extracted folder, XISO, CCI, original game XBE only, or safely preserved full-disc ISO where supported; allow multiple independent outputs. Patch artifact export (JMP/IPS/BPS/Xdelta/EIPS/mpdpack) is a separate concern. Native XISO rebuilding/folder extraction and unified GUI routing remain unimplemented. Use Repackinator's documented CLI as a reference, not as a silently bundled dependency; review its GPL-3.0 license. Do not silently omit multi-file changes for XBE-only output, corrupt full-disc layouts, or break the mandatory updater. Add round-trip tests before marking support complete.
