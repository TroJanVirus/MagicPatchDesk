## v0.14.0-test.8 release evidence — 2026-11-22

Preview published successfully at https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.8 (canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` present). The scoped GitHub Actions verification completed **success**: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44985163278. Verified updater protocol, PatchHub media-index generator fixture, native JMP icon embedding, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, native Windows x64 cross-build, PE header, `core.AppVersion`, SHA256SUMS and publication. No new always-on Actions workflow was introduced.

**Still unverified:** hands-on Windows 10/11 light-theme visual contrast and UI clicks, 1024×720 interactive layout, first-launch network sync, actual PatchHub media-index deployment or full website screenshots, and Xbox hardware. Native Windows `--mpd-ui-smoke-v1` assertions were extended but could not be executed on the Linux verification runner. The original web host must serve `patchhub-media-v1.json` for non-enumerable arbitrarily named media.

## 2026-11-22 — v0.14.0-test.8 candidate (new owner request, verification pending)

- Owner reports Light mode has unreadable navy panels/text: redesigned Settings into a dedicated **Appearance** section with four preview swatches and direct persisted theme selection. Introduced a coordinated high-contrast light palette, legacy GDI colour adaptation and window title-bar mode switching. Added 1024×720 Windows UI smoke expectations and theme settings migration tests.
- **Check + sync** bundles app release checking, GitHub JMP and PatchHub website metadata refresh. The single **Check at Start** preference controls all automatic network sync; when disabled, local catalogue still loads and manual controls remain. Combined sync is sequenced to avoid concurrent full catalogue reads.
- Root causes of missing comparison images identified in source: directory derived from GitHub JMP filename, early return on ambiguous XBE revisions before reading manifest, site game cards omit per-patch image filenames, index may be absent, and silent lookup/download errors. New index-first lookup matches game/TitleID/region/known revision/normalised patch identity independent of website folder spelling; bounded filename probing is only fallback. UI now exposes **Why no images?** with errors and logs failures. Genuine cross-region matches remain forbidden.
- Critical upstream-versus-site discrepancy: website AMF Xtreme Bowling **NTSC Widescreen [Esppiral]** media does not have a matching GitHub JMP in the current repository; GitHub has Esppiral **PAL Widescreen**, plus a separate NTSC 720p patch. Do not attach screenshots across patch identities. Arbitrary filenames across all games still require PatchHub host deployment of `All-Patches/patchhub-media-v1.json` generated from server files, as documented in `docs/RELEASE_NOTES_v0.14.0-test.8.md`.
- Candidate v0.14.0-test.8 increments `core.AppVersion`, retains immutable updater protocol v1, canonical `MagicPatchDesk-Windows-x64.exe` + `SHA256SUMS.txt`, portable/managed targets and user data. Await Go tests, race detector, vet, icon generator, release-contract check, Windows x64 cross-build and published release evidence before saying the new preview is available. No physical Xbox or real interactive Windows QA has been performed.

## 2026-11-22 — v0.14.0-test.7 candidate (release QA pending)

- Owner reports no recent crashes; CTD work is deprioritized, not asserted resolved.
- Added **Light** as a fourth persisted theme alongside Midnight/Forest/Plum.
- Replaced per-game screenshot URL seeds with generic discovery from `patchhub-media-v1.json` (exact patch-folder validation), plus bounded directory listings/HEAD/range-GET fallback. **The web host must publish its real file index** using `scripts/build_patchhub_media_index.py`; absent that index, arbitrary names cannot be fully discovered by guessing. No production AMF special case.
- Catalogue now records every collapsed semantic duplicate pair and offers a Settings **Duplicates · View** private text report. One exact-identical upstream source pair confirmed: Jao12345's two GoldenEye - Rogue Agent JMP paths; other duplicate pairs require a live catalogue audit.
- User explicitly requests a v0.14.0-test.7 update. AppVersion bumped, updater protocol v1, canonical EXE and SHA256SUMS remain unchanged. Publication requires CI tests, vet, Windows cross-build and checksums. Any CI failure blocks release.
- Preserve v0.14.0-test.6 as rollback. Do not report light theme or dynamic image delivery as hands-on Windows validated without field QA.

## 2026-11-22 — CRC warnings, image discovery, theme and converter follow-up (development branch; NOT field verified)

- Main Patch Library output preflight now compares the exact original `default.xbe` CRC32 against selected JMP constraints before ordinary export confirmation. A mismatch produces an explicit warning and optional one-operation override, without persisting unsafe bypass settings. The selected patch footer displays expected versus loaded CRC in red when mismatched.
- Patch list count is deduplicated entries while sync reports downloaded source files (e.g. 800 versus 803 can reflect three collapsed entries); exact duplicate identities still require a live catalogue audit.
- Convert Patch permits a selected script alongside an optional known pre-patched XBE. With the original source supplied, the script output is compared byte-for-byte to the reference before saving converted output; the converter's existing internal round-trip check still applies. This is single-file verification, not proof of multi-file/PF equivalence.
- Settings adds persisted Midnight, Forest and Plum palettes. PatchHub media lookup adds a small GET range fallback for HEAD-blocking servers, known owner-provided AMF Bowling URL (and existing Phantom Crash URL), and displays only successfully downloaded/validated images. Live AMF HTTP delivery and actual GUI display are not verified.
- This branch has NOT been compiled, tested on Windows, or released. Do not mark as shipped or close the prior CTD issue. Avoid unnecessary Actions churn; run tests, vet, Windows cross-build, updater checks and real GUI QA before merge/release.

# MagicPatchDesk — development handoff

> **Current-state notice (2026-11-22):** This file is an **append-only development history** spanning early prototypes through v0.14.0-test.5. Several old "known gaps" and "planned" paragraphs below were implemented later. For the authoritative current state, open **[PROJECT_CONTEXT.md](PROJECT_CONTEXT.md)** and **[docs/STATUS.md](docs/STATUS.md)** first; for decisions made by the owner, read **[docs/DECISIONS.md](docs/DECISIONS.md)**. Check `main` and live Releases for newer changes. This file must not be used by reading only its opening sections.

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

## Historical known gaps (early build; superseded by later entries)

- Windows UI and console-boot compatibility have not been independently validated on real hardware.
- Xdelta3 secondary compression and custom VCDIFF instruction tables are unsupported.
- Full-disc ISO expansion is restricted; arbitrary files, compressed images, and ISO repacking are unsupported.
- No GPU patch acceleration; worker count applies to indexing and ISO copying, not sequential JMP records.
- No automated game-image regression corpus (copyright concerns); add synthetic XDVDFS and JMP tests as improvements are made.
- No executable auto-updater, presets/undo journal, batch conversions or complete game-asset editor yet.

## Historical next actions (early build; use PROJECT_CONTEXT.md for current priorities)

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


## 2026-11-22 v0.14.0-test.1 published and validated

**Published prerelease:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.1

**QA:** one-shot Actions https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44921737286 **SUCCESS**, including Linux Go suite and race detector, vet, Win64 cross-build, native Windows Go tests, SHA256 verification of the exact release candidate, existing `--mpd-healthcheck-v1`, and `--mpd-ui-smoke-v1` painting all five Game Manager views. Canonical EXE SHA256 is `86ba4f82d64b858309dcc181554d7ad20152b5009f6f4c2967cb01a83089720e`. `SHA256SUMS.txt` also uploaded. PR #12 merged to main: https://github.com/TroJanVirus/MagicPatchDesk/pull/12. One-shot release workflow removed from main; manual-only verification kept. No recurring Actions cost introduced.

**Open limitations:** native Windows mouse, folder picker and GitHub confirmation flows need interactive human QA; xemu/physical Xbox boots and actual retail images are NOT verified. The confidence JSON index intentionally begins empty; user-submitted GitHub Issues are unreviewed until the maintainer reviews, deduplicates and commits them into the versioned index. Do not claim there are verified confirmations before real field reports exist. Update-check sync of reviewed confidence index is automatic, but submission is explicit and consented. No numeric confidence score.

## 2026-11-22 — v0.14.0-test.2 physical hardware field reports

User-directed improvement to the **Game Manager → FIELD REPORTS** workflow. Real Original Xbox confirmations now capture independently: motherboard revision (v1.0, v1.1, v1.2, v1.3, v1.4, v1.6, v1.6b, or an explicit **I don't know**), RAM upgrade status (No upgrade / Upgraded / I don't know), and CPU upgrade status with the same three choices. Hardware details start at **unknown**, not assumed stock, and reset when selecting another game. The test environment remains one of **xemu** or **physical hardware**. No Xbox hardware information is required or submitted for xemu tests.

Typed, allowlisted fields: `xbox_revision` (`unknown` or one listed revision), `ram_upgrade` and `cpu_upgrade` (`unknown`, `stock`, `upgraded`). The app includes them in the explicit public GitHub Issue consent preview, report payload and displayed per-patch confidence entry. Source game bytes, paths and patch payloads remain excluded. The existing reviewed `magicpatchdesk-confidence-v1` schema remains readable: older reports missing all fields are correctly labelled **Not recorded**, not incorrectly treated as a tester answering **I don't know**. Partial triples or physical hardware values in xemu reports are rejected.

The launch/healthcheck and `--mpd-apply-update-v1` contract, canonical release EXE and SHA256 manifest requirements are unchanged. External revision reference credited in `CREDITS.md` (ConsoleMods Wiki, documentation only). This is self-reported hardware information; no runtime console detection has been implemented. Automated Go, Win32 and GUI paint QA plus real Windows field testing must be differentiated.


## 2026-11-22 — v0.14.0-test.2 published, native QA green

**Release:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.2
**Merged source:** https://github.com/TroJanVirus/MagicPatchDesk/pull/13 (merge SHA `cefab78b7e2648904392b42d1fc7fe6e466a7b87`)
**Gate:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44923406249 (**SUCCESS**, all Linux, Windows and publish jobs).

Completed the precise real-console compatibility fields requested by the user: revision (including explicit unknown), RAM upgrade and CPU upgrade each allow stock/upgraded/unknown independently. The selected fields are displayed in the report history, submitted GitHub Issue and public consent dialog. Emulation-only reports omit all physical-console data. Legacy index records missing fields remain valid and marked **Not recorded**. New core tests exercise combinations, source-safe issue body, xemu isolation and old JSON. Windows native UI smoke ensures all new buttons are clickable at 1024×720 window bounds.

The published unsigned Windows EXE `MagicPatchDesk-Windows-x64.exe` SHA256 is `1de2630882342e57bfb9958c3cb8bdfc46fb74d93732ade159e695d0d48d427f`. `SHA256SUMS.txt` is published alongside the EXE. v1 self-update and healthcheck contracts are unchanged. Transient CI run #44923083189 failed due to an overstrict smoke-test row bound that mistakenly tested the existing confirmation row against the new selector-row range; the assertion was fixed in subsequent run #44923406249. The consumed release workflow was removed from main to avoid recurring Actions cost.

**Not verified:** actual users clicking the native Windows dialogs, physical Xbox/Cerbios bootability or any independently confirmed hardware report. Field-report contents are user-supplied claims; public GitHub Issues still require maintainer review before inclusion in the curated confidence index.


## 2026-11-22 — v0.14.0-test.3 duplicate-safe JMP import and multiple PatchHub images

**User requirements delivered:** Import JMP now detects an **exact same byte-content** JMP/PF anywhere in the Custom or GitHub patch library, leaves it completely unchanged and shows the nonmodal bottom status message **"Duplicate patch found! No need to add."** in red, plus an event in the local verification log. Distinct patch bytes sharing the same filename are preserved as an independent numbered import, never overwritten. The drag/drop file import path follows the same safety rule. Implementation: `core/import_guard.go` and `core/catalog.go`; Win32 actions/status `cmd/magicdesk/main_windows.go` and `paint_windows.go`.

**Patch images:** `core/patchhub_gallery.go` adds separate per-patch before/after screenshots from public PatchHub HTML-linked image URLs and from imported PatchHub `titles/` patch folders. Each screenshot must match a single patch identity; mismatches/ambiguity are not guessed. `HubIndex` now retains optional per-game screenshot metadata with backwards-compatible `gallery_scan_version`. `CatalogPatch.Gallery` associates matched evidence with a specific patch. `cmd/magicdesk/gallery_windows.go` provides a native multi-image, side-by-side viewer and on-demand offline cache through the trusted HTTPS-only artwork downloader; `art_windows.go` preserves photo aspect ratio. `View images (N)` appears only where photographs are discoverable.

**Release verified:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.3 — unsigned developer prerelease. Canonical Windows EXE SHA256 `9ddc49f889b86ffca10ef13e5feaa8b0bac351493970ce43d71dd6d02374ea60`, with `SHA256SUMS.txt` published. Latest successful full release gate: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44971296368 — Linux Go tests/race/vet, native Windows unit tests, EXE SHA check, updater healthcheck and multi-image gallery 1024x720 GUI paint/navigation smoke all **passed**. The first one-shot attempt failed only due Go regexp's maximum repeat bound, fixed before this passing run. The transient publisher workflow was removed afterward to preserve GitHub Actions minutes.

**Important unresolved evidence:** The public GitHub JMP repository does not include comparison pictures. The program can use image URLs actually embedded in PatchHub's page metadata and local imported screenshot folders; it cannot automatically know private paths not published by the website. A real live PatchHub screenshot sample and hands-on Windows file dialogue/gallery testing are still required. Do not claim every PatchHub patch has usable online gallery links solely from synthetic test fixtures. See `docs/JMP_IMPORT_AND_PATCHHUB_GALLERY.md`, `docs/STATUS.md`, and `PROJECT_CONTEXT.md` for the current handoff and next steps.


## 2026-11-22 — v0.14.0-test.4 crash resilience and native UI repairs in progress

Owner report: adding a Phantom Crash duplicate previously preceded an unlogged crash during XBE selection; an exact-byte-only import guard still permitted duplicates; Convert & Create had irrelevant IPS password/RLE controls; patch notes showed literal <br> and truncated instructions; scrolling patch records also scrolled the patch list. Direct source examination confirmed the reporter only covered handled errors, folder-import bypassed duplicate checks, the notes footer used strLimit(...,93), and WM_MOUSEWHEEL changed both offsets.

Implementation staged on branch fix/crash-report-dedup-notes-scroll-v014: core/crash_guard.go plus Win32 capture and next-launch report integration, raw private evidence and sanitized GitHub metadata; core/jmp_equivalence.go and catalogue deduplication across files/import paths; core/jmp_notes.go + native read/copy full-note viewer; per-region scrollbars; conditional IPS/EIPS controls. New Go tests core/crash_dedup_notes_test.go and native UI smoke added. The previous Phantom Crash fault is *not independently diagnosed* (no prior trace). Microsoft WER documented native dumps require admin-level HKLM configuration, **not** silently installed by the app. Full reference docs/CRASH_RECOVERY_AND_UI.md. Await gated CI and release validation; do not claim current feature has passed until actual results are checked.


## 2026-11-22 — v0.14.0-test.4 published and merged (crash reporting and Patch Library)

**Published preview:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.4
**Merged PR #16:** https://github.com/TroJanVirus/MagicPatchDesk/pull/16, merge SHA `8a4dcadc40e92a1a2044c2ff489f633a9b4d4ea6`.
**All-green release gate:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44973558561 — Linux Go unit/race/vet, Windows x64 build, Windows Go tests, exact EXE SHA256 + v1 updater healthcheck, Windows native UI smoke (full formatted JMP notes, separate patch/record scrollbars, format-specific converter controls) and prerelease verification.
**EXE SHA256:** `5cb91f4afe17ca45efe5593b8c004c1f2873fab3455629e37adea3da186ced3b`; canonical Windows executable and SHA256SUMS.txt published; updater v1 unchanged, one-shot release workflow deleted after success.

The user's specific crash **was not** reproduced or diagnosed (no trace from the old build). The old reporter was confirmed to only capture handled errors. New source creates durable process markers, private log (including caught panic stack where available), next-launch unclean-exit sanitized JSON and persistent red notice; Settings Crash logs and View crash log show private raw evidence, while GitHub issue submission only uploads whitelisted safe summaries. No automatic Windows-native minidump/registry write; WER HKLM opt-in is documented. A forced kill is an unclean exit, not proof of software bug.

Semantic JMP dedup now compares source identity, ordered actual record edits, targets/groups and states; Import JMP, drag/drop and folder import avoid equivalent duplicates despite author/notes/BOM/CRLF differences, and preexisting duplicate catalogue entries collapse without deleting source files. Full notes have safe HTML-break/list/entity rendering, word-wrapped native viewer and Copy all. Right-side lists have independent scrollbars targeted by hovered pointer; Convert & Create displays IPS/EIPS controls only when relevant.

**Outstanding manual QA:** real Windows user test selecting a Phantom Crash XBE with old duplicates, actual abnormal termination private evidence and sanitized report upload, GUI notes and thumb dragging at 720p, plus live PatchHub screenshot coverage. All automated tests passed but **not** actual Xbox game boot/certification. Full design/implementation source and tests: `docs/CRASH_RECOVERY_AND_UI.md`, `docs/RELEASE_NOTES_v0.14.0-test.4.md`, `core/crash_dedup_notes_test.go`, `cmd/magicdesk/ui_smoke_windows.go`.


## 2026-11-22 — verified v0.14.0-test.5: Patch Library output integration and portable updater

**Owner request:** Remove separate Game Outputs navigation and make Patch Library Apply require an explicit output selection; allow multi-select plus Preserve current format; rename Game Manager to Batch Process Library; anchor Settings/About at bottom; merge Channel & Storage into Check for Updates → Update Settings and explain Previews ON; hide all reporting controls except Connect until GitHub token connected; fix portable update refusal when an independent managed install exists.

**Shipped source:** `core/output_choices.go` validates source kind and explicit format selection. `cmd/magicdesk/library_outputs_windows.go` creates a modal over Patch Library with five independent output types and Preserve current format. `cmd/magicdesk/paint_windows.go` renders a truly disabled/non-hit-tested Apply Patch until source, patch and valid output are chosen; `main_windows.go` routes Apply into existing independently staged `startGameOutput` and resets output choices on source change. `game_outputs_windows.go` retains `core.ExportGameOutputs`, preserving no-overwrite and identity checks; no separate Game Outputs sidebar link. Batch Process Library remains a distinct bulk workflow. Settings/About bottom anchor, Maintenance Check for Updates and nested Update Settings, and only Connect until Credential Manager confirms connection implemented in Win32 paint/auth code.

**Updater root cause:** the former portable `updateNow` branch refused whenever `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe` existed; matching filenames were NOT the issue. The fixed updater targets the *executing* EXE, not another installation. `core/update_intent.go` adds a locally stored PID/target/staged-SHA256 intent required by the unchanged five-arg v1 helper for portable targets; new staged binary and helper run `--mpd-healthcheck-v1` before replacement. Managed updates retain legacy fixed path and rollback; portable updates use `<stem>.previous.exe` in the same folder. Both leave the other's EXE intact. A user on the older blocked portable build needs one **manual download of v0.14.0-test.5** to acquire this new updater code, and portable folder must be writable.

**Release verified:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.5 — exact Windows EXE SHA256 `76d1edb3da73cb7a9896c3938820f48802cd4d2237985571411a10545f1c19a4`, SHA256SUMS.txt attached. Complete Linux/race/vet, Win64 build, native Windows unit, exact EXE healthcheck and UI smoke, and publisher gate **all passed** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44976120121. Source PR #18: https://github.com/TroJanVirus/MagicPatchDesk/pull/18. One-shot publishing workflow removed to save CI minutes.

**Remaining manual QA:** live Windows 10/11 portable self-replacement when a managed copy exists, actual game-output format selection/clicks on small screens, extracted game/XISO/CCI/full-disc export and preserving originals, physical Xbox game boots, and Phantom Crash original-XBE past failure (issue #17). Synthetic/unit and Win32 UI smoke are NOT confirmation of user-system update swaps or retail game compatibility. Canonical new-conversation entry `PROJECT_CONTEXT.md`, `docs/STATUS.md`, `docs/PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md`, `docs/UPDATE_CONTRACT.md` and `docs/DECISIONS.md`.


## 2026-11-22 — real-world v0.14.0-test.5 startup CTDs, root cause unresolved, DO NOT CLAIM FIXED

User downloaded/runs v0.14.0-test.5 and reported two CTDs in a row, supplied `crashes.zip` privately. Two previous-session records began 2026-11-22T02:31:42Z and 02:31:45Z (NZDT ~15:31). First runtime log contains a Windows native `Exception 0xc0000005` (read access violation), fault PC `0x7ffb695e1548`. The visible Go call stack ends at `syscall.(*LazyProc).Call` from `main.initApp():483`, the normal `DispatchMessageW` message loop; the actual faulting Windows DLL/callback is missing from the Go trace. Second crash's runtime stderr is empty, so cause cannot be identified or assumed same. Third active session marker in ZIP; no reason to delete caches, JMPs or user data. Both marker stages are `startup` even while in the event loop, showing stage tracking lacks Win32 paint/dispatch events.

**Issue #19:** https://github.com/TroJanVirus/MagicPatchDesk/issues/19 (sanitized public findings, not raw user logs). **Need** Windows Event Viewer → Windows Logs → Application → Application Error event or `perfmon /rel` for 2026-11-22 ~15:31 NZDT: **Faulting module name and fault offset**. Do not claim GDI+, background sync or duplicate JMP are the cause without evidence. No speculative replacement release built. Update PROJECT_CONTEXT.md / docs/STATUS.md flags as release blocker notwithstanding the previously green 44976120121 test 5 synthetic CI. Future agent must prioritize real Windows startup stability, safe native UI diagnostics and reproduction before more features or claiming a fixed release.


## 2026-11-22 — v0.14.0-test.6 officially published: owner JMP icon, exact UI labels, patch-specific media index

**Published validated unsigned preview:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.6; native Windows EXE `MagicPatchDesk-Windows-x64.exe` SHA256 `639747d63f6deb63add495d352fb8fdd182284a511d8c23979d9c2759434c9a7` and `SHA256SUMS.txt`. **All successful Linux, race, vet, Win64 build, Python media index/icon generator and native Windows Go/EXE health/UI smoke:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44979391498. Earlier failed CI runs (regex compilation errors) published NOTHING; repaired before success. One-shot publisher removed to minimize Actions minutes. **Source PR #20** https://github.com/TroJanVirus/MagicPatchDesk/pull/20.

Owner requested sidebar text **Convert Patch** and **Batch Process**, exact Win32 window title **Magic Patch Desk | Powered by Patch Hub**, and the uploaded transparent Jay's Magic Patch logo **inside the EXE**. `assets/jmp-logo.ico.b64` stores text-encoded actual icon and `scripts/build_windows_icon.py` emits AMD64 COFF .rsrc for RT_GROUP_ICON & RT_ICON, ignored .syso build intermediate. Native EXE GUI smoke validated caption & LoadIconW(resource1). No other asset needed next to EXE.

**PatchHub gallery:** real user URL `https://www.jayxbox.com/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Phantom%20Crash%20%7BWidescreen%7D%20%28NTSC%29%20%5BJay%5D/1.%20Before.jpg` now parses as exact Phantom Crash USA v5 JMP `Before #1` (formerly rejected). Also handles arbitrary-named JPG/PNG/GIF inside an exact game/version/TitleID/patch folder, multiple comparison images, metadata cache refresh version2, trusted downloads and offline cache. Async bounded lookup only for selected patch, first tries HTML directory-listing links then up to 20 conventional HEAD candidates. A **complete** media index `https://www.jayxbox.com/All-Patches/patchhub-media-v1.json` is supported by `core/patchhub_media_discovery.go` and generated from actual website image files by `scripts/build_patchhub_media_index.py`. **Website JSON has NOT been published.** The GitHub app cannot deploy to jayxbox.com, so arbitrarily named photos remain undiscoverable on a server with directory listing disabled; no claim all images are available. See `docs/PATCHHUB_MEDIA_AND_BRANDING.md`, `docs/RELEASE_NOTES_v0.14.0-test.6.md`, and `core/patchhub_media_discovery_test.go`.

**Crash status unchanged:** user tried `perfmon /rel` and found no Windows crash event corresponding to both CTDs on v0.14.0-test.5. App private runtime log showed native Windows 0xC0000005 while DispatchMessageW was running, but no faulting DLL/callback. The second abnormal exit has no stack. `markCrashStage("ui_callback")` now runs before normal message loop to improve future evidence; **not a CTD fix**. Keep issue #19 https://github.com/TroJanVirus/MagicPatchDesk/issues/19 OPEN and do not ask for an absent reliability monitor event again. Real user Windows startup and photo display testing remain pending; native CI only proves bounded synthetic/regression paths.
