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

See `docs/IMAGE_OUTPUT_WORKFLOW.md` and GitHub issue #9. An XISO/CCI/ISO/folder/XBE input must not lock the *game* output format. Let users explicitly select patched extracted folder, XISO, CCI, original game XBE only, or safely preserved full-disc ISO where supported; allow multiple independent outputs. Patch artifact export (JMP/IPS/BPS/Xdelta/EIPS/mpdpack) is a separate concern. Native XISO rebuilding, folder extraction and the unified native Game Outputs workspace were subsequently implemented; see the 2026-11-22 status below. Use Repackinator's documented CLI as a reference, not as a silently bundled dependency; review its GPL-3.0 license. Do not silently omit multi-file changes for XBE-only output, corrupt full-disc layouts, or break the mandatory updater. Add round-trip tests before marking support complete.


## 2026-11-22 implementation continuation: Game Outputs + credits

The main branch now contains \`core/game_outputs.go\` (XISO extraction preserving empty directories, game-folder copy, XDVDFS directory writer, verification, single/multi-format export), \`core/game_outputs_test.go\` synthetic regression tests, \`cmd/magicdesk/game_outputs_windows.go\` (patch-aware game conversion UI controller), and a new native Win32 **Game Outputs** workspace. \`core/xiso.go\` exposes directory paths; \`core/settings.go\` persists last output selections per source class. \`CREDITS.md\` records specific reference roles and licences; About lists major references and links to credits. Repackinator (GPL-3.0) was consulted as a workflow reference; its source was not incorporated. PF inputs are staged as full folders for eligible conversions; incompatible XBE-only exports reject multi-file operations.

**Headless test outcome (2026-11-22):** [GitHub Actions run #44913720896](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44913720896) checked out `c119e657599297f3fa350c8246c3c40bb5b4719c` with Go 1.23.12 and passed `go test ./...`, `go test -race ./core`, `go vet ./...`, and Windows x64 GUI cross-build (`CGO_ENABLED=0`). The verification workflow was switched to manual-only `workflow_dispatch` after the run, avoiding automatic Actions churn. **Still not tested:** Win10/11 GUI interactions, xemu, Cerbios/physical console boot, in-app updater or installation. No executable was published from this check. Keep the update contract and issue #9 open until runtime validation.


## 2026-11-22 preview publication: v0.13.0-test.1

Published [v0.13.0-test.1](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.13.0-test.1) as a **prerelease only**, with updater-compatible canonical asset `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`. Release tag points at `1271edf10daa2ce7dc339df01ee4f4c5eabf8f9f`. The [headless release run](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44914374173) PASSed: `scripts/verify_release_contract.sh`, full `go test ./...`, `go test -race ./core`, `go vet ./...`, Windows x64 cross-build, **native Windows** `go test ./...`, exact candidate SHA256 verification on Windows, and executed `--mpd-healthcheck-v1` Windows GUI EXE returning exit 0. The GitHub release API confirms uploaded assets/digests. The post-publish `gh release view --json prerelease` verification command failed solely because GitHub CLI expects the field `isPrerelease`, so the overall workflow's UI says failure even though the release published and verified via the GitHub API. The one-shot publishing workflow was removed after publication to prevent repeat Actions consumption; manual `.github/workflows/one-shot-verify.yml` remains available. GUI functional testing, real Xbox boot, Cerbios CCI loading and in-place self-update/rollback are NOT yet validated. Preserve v1 updater protocol on next change.

## 2026-11-22 v0.13.0-test.2 follow-up audit

After the previous v0.13.0-test.1 preview, new failure-mode testing identified a **reproducible Go panic** in `core/cci.go` CCI split discovery if unsuffixed `.cci` was in a folder with split parts. Fixed slicing and added split gap/ambiguity regression tests. `core/game_outputs.go` now reads named sidecar patch targets via case-insensitive safe resolution; verifies source/target file byte counts, SHA-256 and all XDVDFS directories, including empty folders; rejects Windows-reserved aliases; writes balanced XDVDFS tree pointers plus 0xFF entry padding and 64KiB image length alignment. `core/game_integrity_test.go` and `core/game_balanced_test.go` cover those invariants, including 513 members in a folder and split CCI roundtrips. `cmd/magicdesk/game_outputs_windows.go` protects sidecar-only pack TitleID/CRC verification, supports Skip patch and `.xiso` browsing. `paint_windows.go` scales Game Outputs for ~720p displays. New test-only `--mpd-ui-smoke-v1` renders all native tabs and verifies Game Outputs click surfaces in a 1024×720 outer window with no network/disk mutation. The self-updater's v1 helper flags and canonical release asset are unchanged; version bumped to `v0.13.0-test.2`.

[Reference credits](CREDITS.md) now explicitly identify Andrew de Quincey's Xbox DVD filesystem document and purelogiccode/XISOStudio technical docs, reference only. **Hardware bootability, real user GUI interaction and in-app update/rollback remain unverified** until manually tested. Only publish this version as prerelease after gated Linux/Windows tests, race, vet, updater-contract and native UI smoke checks succeed. The publishing workflow is ONE-SHOT and must be deleted immediately after publication to prevent Actions churn.


## 2026-11-22: v0.13.0-test.2 automated release validation and remaining field QA

- **Published preview:** [v0.13.0-test.2](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.13.0-test.2), SHA-256 for canonical `MagicPatchDesk-Windows-x64.exe`: `6fcf1a9970c934d124521f6bd8b5bf76d0ae004fbb8df2ca5401880ddb583543`.
- **Gated Linux/Windows pipeline:** [GitHub Actions run #44916327760](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44916327760) **SUCCESS**. Verified `go test -count=1 ./...` on Linux and native Windows, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 GUI cross-build, updater CLI contract, SHA256SUMS before/after artifact transfer, exact Windows release candidate execution with `--mpd-healthcheck-v1` and `--mpd-ui-smoke-v1` (all workspaces, export hitboxes at 1024×720). Published canonical EXE and SHA256SUMS after all gates passed, then verified both uploaded assets in GitHub. [Fix PR #11](https://github.com/TroJanVirus/MagicPatchDesk/pull/11) was squashed to main.
- **Actions conservation:** one-shot publisher removed after success, leaving only the previously established manual-only verification workflow. No recurring CI or automated diagnostics upload was added.
- **Real bugs remedied:** panic from CCI standalone+split names, split sequence omissions/ambiguity, incomplete extraction/copy checks, missing XDVDFS empty-folder/hash verification, unbalanced XDVDFS directory search, missing directory padding and 64KiB output alignment, case-sensitive named patch targets and unchecked sidecar-only pack identities, missing explicit Skip patch, clipped Game Outputs panel at 720p.
- **Still outside automated verification:** interactive mouse/keyboard usability on actual Windows 10/11 machines, full installed v0.12→v0.13 updater and rollback under a user session, real XISO/CCI game boot in xemu and physical Xbox/Cerbios, unusually large FATX-sized split sets and every retail game's quirks. Do not assert these work without field testing; Issue #9 stays open. The build is **prerelease, not stable**.


## 2026-11-22 — v0.14.0-test.1 Game Manager and field confidence design

Scope follows explicit user decisions: visual scrollable integrity log rather than a large static preflight wizard; folder-based discovery of extracted/XISO games, per-title patch choices and opt-in unattended processing with uncertain cases deferred; JMP hex overlap visualization plus independently verified JMP merging; read-only image browser; user-submitted compatibility events separated by xemu versus real Original Xbox; every patched output remains an independent copy. No new console transfer, emulator execution, CLI or automatic modification of original game files.

Implementation: `core/library_scan.go` catalog matches by exact Title ID and XBE CRC32; title-only suggestions require human review; CCI is review-first because native per-member metadata is not yet directly indexed. `core/batch.go` creates new numbered packages with independent output formats and rechecks source XBE CRC/TitleID plus patch SHA256 immediately before writing; review/skip jobs make no changes. `core/verification_log.go` provides bounded thread-safe event records and source-relative conflict hex windows; `core/merge_jmp.go` requires explicit conflicting-byte authorization and re-applies output to verify. `core/image_explorer.go` lists XDVDFS files/dirs and allows verified individual extraction. `core/confidence.go` validates reviewed strict JSON and atomically syncs `confidence/reports.v1.json`; optional `core/confidence_submit.go` sends a consented issue with hashed patch identity and no game bytes. `cmd/magicdesk/library_manager_*_windows.go` implements GUI modes GAMES / IMAGE FILES / HEX OVERLAPS / VERIFY LOG / FIELD REPORTS; the ordinary update-check flow syncs reviewed confidence metadata separately. The v1 self-update CLI contract is unchanged. See `docs/LIBRARY_MANAGER_AND_CONFIDENCE.md` for full technical and moderation details.

**Important:** submitted GitHub Issues are unreviewed input and are NOT automatically promoted into confidence data. A maintainer must review and commit revised `confidence/reports.v1.json` with higher revision; the next app update-check fetches it even if the EXE is current. The index contains individual dated reports, not percentage ratings. Both user consent for submission and source-patch SHA256 matching are mandatory.

**QA still needed:** one-shot Go tests, race/vet, Win32 cross-build, native Windows tests and GUI smoke, then user-driven Windows 10/11 and Xbox/Cerbios validation. Do not claim real console or general compatibility based on synthetic images or emulator assumptions. No recurring Actions workflow should be installed.
