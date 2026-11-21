# MagicPatchDesk — CURRENT PROJECT CONTEXT

**Canonical entry point for a new conversation, a new AI coding agent, or a new contributor.** The repository (especially `main`) is authoritative. This document is the compact current-state handoff; `AI_HANDOFF.md` and `docs/PROJECT_STATE.md` preserve *dated development history*, including old plans that may now be complete. Do not mistake an early section of those files for the present status.

**State last verified:** 2026-11-22 NZ. Before making changes, retrieve **live** `main`, `core/update.go`, the latest GitHub Release, relevant GitHub Issues and recent runs; this snapshot is a baseline, not a permanent assertion that the tag and HEAD are unchanged.

## How to resume in a fresh ChatGPT conversation

Start with this one request:

> **Continue MagicPatchDesk from its GitHub repository: https://github.com/TroJanVirus/MagicPatchDesk . Read AGENTS.md, PROJECT_CONTEXT.md, docs/STATUS.md, docs/DECISIONS.md, AI_HANDOFF.md and docs/PROJECT_STATE.md; inspect the latest main branch, release, open Issues and code before making changes. Treat the repo as authoritative, preserve the updater and original files, and avoid unnecessary GitHub Actions runs. Pick up the latest outstanding work.**

An AI agent does **not** need access to any previous chat to recover the implementation, previous decisions or known QA gaps. It **does** need permission/access to read this GitHub repository and current Issues/Releases. If those are unavailable, report the access blocker rather than guessing, and ask to connect the relevant GitHub account/repository.

### Reading order (do not skip)

1. **`AGENTS.md`** — nonnegotiable coding, security, test, privacy, update and licensing constraints.
2. **`PROJECT_CONTEXT.md`** (this document) and **`docs/STATUS.md`** — latest state, release evidence, unverified areas and immediate next actions.
3. **`docs/DECISIONS.md`** — decisions the project owner has already made, including explicitly deferred features; never ask the same question again without reason.
4. **`AI_HANDOFF.md`** and **`docs/PROJECT_STATE.md`** — implementation chronology and previous agent handoffs; later dated entries supersede earlier ones.
5. **`README.md`** — end-user behavior, build instructions, feature explanations.
6. **`docs/UPDATE_CONTRACT.md`**, **`CREDITS.md`** and task-specific `docs/*.md`; inspect actual source and tests for final truth.
7. Latest `main` HEAD, GitHub Releases, Issues/PRs and CI run results. Do not infer a future release from version strings or assume that a passing test confirms Xbox game bootability.

## Verified release and repository snapshot

| Item | Status as of 2026-11-22 NZ |
|---|---|
| Primary repository | https://github.com/TroJanVirus/MagicPatchDesk |
| Development source | `main`, native Go and Win32; **not** the older HTML app |
| Latest published release at this snapshot | **v0.14.0-test.2**, an **unsigned developer prerelease**, https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.2 |
| Published canonical Windows executable | `MagicPatchDesk-Windows-x64.exe`; SHA-256 `1de2630882342e57bfb9958c3cb8bdfc46fb74d93732ade159e695d0d48d427f` |
| Required published manifest | `SHA256SUMS.txt` in the same release |
| Latest feature merge | PR #13, https://github.com/TroJanVirus/MagicPatchDesk/pull/13, merge `cefab78b7e2648904392b42d1fc7fe6e466a7b87` |
| Release verification | https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44923406249 — Linux unit/race/vet, Win64 build, native Windows unit tests, EXE SHA-256, `--mpd-healthcheck-v1` and `--mpd-ui-smoke-v1` all **passed** |
| Public patch-confidence index | `confidence/reports.v1.json`: revision 1 and **no reviewed user reports** at this snapshot |
| Open field-QA/format issue | #9, https://github.com/TroJanVirus/MagicPatchDesk/issues/9 — its original request text predates features implemented in v0.13/v0.14; read subsequent issue comments before deciding what is still open |
| CI policy | Only manual `.github/workflows/one-shot-verify.yml` retained; one-time release publisher workflows removed to conserve GitHub Actions time |

**Difference between source and release:** Additional documentation-only commits can follow a release on `main` without changing the tested EXE. Compare code/tree changes against the release target before claiming a new functional build is needed.

## Product and architecture

**Goal:** An offline-capable **native Windows 10/11 x64 Xbox game/patch workbench** with patch discovery, safe independent game outputs, patch-format conversion and diagnostics. Language is **Go 1.23+ standard library**, Windows UI is Win32/GDI/GDI+ (not Electron, .NET or an installer). Build with `BUILD-WINDOWS.cmd` or `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w" -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk`.

| System | Ground-truth source / contract |
|---|---|
| Win32 UI, input, routing, workspace views | `cmd/magicdesk/main_windows.go`, `paint_windows.go`, other `*_windows.go`; smoke `ui_smoke_windows.go` |
| Patch formats and verification | `core/jmp.go`, `ips.go`, `bps.go`, `eips.go`, `xdelta.go`, `formats.go`, `preview.go`, `directconvert.go`; JMP author/CRC/disabled-record semantics in `AGENTS.md` |
| Multi-file patches, PF and packs | `core/targets.go`, `pf.go`, `pack*.go`, `xiso_multi.go`; `docs/PF_FORMAT.md`, `docs/PATCH_PACK_V2.md` |
| Game-image I/O and selectable independent outputs | `core/xiso.go`, `cci.go`, `game_outputs.go`, `image_explorer.go`; `cmd/magicdesk/game_outputs_windows.go`; `docs/IMAGE_OUTPUT_WORKFLOW.md` |
| Per-game library scan, batch decisions and cautious execution | `core/library_scan.go`, `batch.go`; `cmd/magicdesk/library_manager_windows.go`, `library_manager_paint_windows.go` |
| JMP overlap, hex visualization, merging, visual log | `core/verification_log.go`, `merge_jmp.go`; Game Manager HEX OVERLAPS / VERIFY LOG views |
| XBE metadata, title/version/CRC32 | `core/xbe.go` |
| Reports and curated confidence events | `core/confidence.go`, `confidence_submit.go`, `report_sync.go`, `diagnostics.go`, `developer_dashboard.go`; `confidence/reports.v1.json` |
| Patch catalogue, metadata and offline cache | `core/catalog.go`, `patchhub.go`, `settings.go`; `README.md` |
| Permanent update protocol | `core/update.go`, `cmd/magicdesk/maintenance_windows.go`, `docs/UPDATE_CONTRACT.md`, `scripts/verify_release_contract.sh` |
| Tests | `go test ./...`, `go test -race ./core`, `go vet ./...`; `core/*_test.go`, `cmd/magicdesk/ui_smoke_windows.go`; `testdata/interop/` |

### Current user-facing workflows

- **Patch Library / Convert & Create:** JMP, IPS, BPS, encrypted IPS, a constrained VCDIFF/Xdelta subset, Rocky5 PF and v1/v2 `.mpdpack` workflows, respecting per-target and source-dependent constraints. Original JMP metadata and author attribution must survive.
- **Game Outputs:** Take original `.xbe`, extracted game folder, XISO/ISO or supported Cerbios CCI source; independently choose safe outputs: patched XBE, independent extracted folder, rebuilt game-partition XISO, CCI, and full-disc ISO only for compatible full-disc sources. Patch-free extraction/conversion is allowed. An XBE alone cannot conjure a complete disc image. Inputs never overwritten; results independently verified.
- **Game Manager:** Scan library folders of extracted/XISO/ISO/CCI games. Show per-game patch options with **Auto / Review / Skip**; only one strict Title ID + original-XBE CRC32 verified single-file JMP may be auto-eligible. Recheck patch SHA256 and game identity immediately before writing new packages. Ambiguous matches and CCI images requiring XBE inspection remain for manual review. Complex games should be surfaced after exact matches.
- **Game Manager → IMAGE FILES:** Read-only image exploration and safe single-member extraction; no forced patch operation.
- **Game Manager → HEX OVERLAPS:** Original and competing hexadecimal byte windows. Conflicting JMP writes must receive an explicit warning and manual approval before a new merged JMP; verify resulting content and preserve provenance.
- **Game Manager → VERIFY LOG:** Bounded scrollable local human-readable log (steps, errors and pass/warning events), not a large preflight wizard.
- **Game Manager → FIELD REPORTS:** Individual dated working/partial/failed entries, **no percent/stars/trust scores**, keyed by patch SHA256 + original XBE Title ID/CRC32. xemu and real Original Xbox testing are **distinct**. Real-console submissions let the reporter choose revision 1.0/1.1/1.2/1.3/1.4/1.6/1.6b/**I don't know**, RAM and CPU each **No upgrade / Upgraded / I don't know**. Historical missing values display **Not recorded**. Physical-console info is never attached to xemu reports. A user must explicitly consent to post a public GitHub Issue; a maintainer reviews and deliberately commits approved reports into `confidence/reports.v1.json`; regular app update-check downloads newly reviewed entries. The index currently has no reviewed confirmations.
- **Maintenance / Settings / Diagnostics:** Managed self-update, offline Custom patch library, optional token-backed GitHub diagnostics, private local reports and a read-only developer issue dashboard. Never upload user game files, binaries, paths, secrets or patch contents.

Read **`docs/DECISIONS.md`** for the durable, explicit product choices and excluded work.

## Critical invariants — preserve in every future change

1. **No in-place game modification.** Patched outputs are **new independent copies**; do not add automatic rollback to replace that model. Never silently drop named-file JMP or PF/pack operations.
2. **Never break updates**: `--mpd-apply-update-v1`, `--mpd-healthcheck-v1`, stable `MagicPatchDesk-Windows-x64.exe` plus `SHA256SUMS.txt`, unchanged program/data directories, an updater helper and previous executable fallback. Follow `docs/UPDATE_CONTRACT.md`, not a novel launcher design.
3. **Offline first and privacy first**: public catalogue/confidence sync is optional, reporting is explicitly controlled; keep private game assets and user paths out of repository and public reports.
4. **Exact compatibility checks**: source CRC32/Title ID, file hashes, whole-output verification; fail closed on suspicious formats, ambiguous matches and unsafe image changes.
5. **Preserve licence/reference attribution** in `CREDITS.md` and the About link. Team Resurgent/Repackinator and other upstream references must never be misrepresented as copied code or endorsement. Check licences before using any code.
6. **Avoid Actions churn**. No always-on or per-commit CI workflow. Run meaningful tests locally or one manually triggered, scoped verification; publish a new release only for validated functional changes or explicit release requests. Never claim booted physical Xbox titles without actual hardware evidence.

## Next priorities and unresolved evidence

**Do not mistake completed implementation for completed field validation.** The v0.14.0-test.2 release passed code, checksum and native Windows headless paint checks; it did *not* establish the following:

1. **Highest priority: hands-on Windows QA** (Windows 10/11): folder-picker and file dialogs, DPI/scaling, all workspace controls, scanned catalogue selection and scrolling, 720p-height UI, multiple per-game output choices, Auto/Review/Skip, manual approval dialogs, no-overwrite and insufficient-disk-space behavior.
2. **Real game/image QA** against legitimately supplied source files: XISO/CCI split sets, extracted folders, named-file JMP, PF/pack conversion, full-disc refusal cases and independent output byte/hash verification.
3. **Game boot QA:** xemu and actual Original Xbox/Cerbios; differentiate emulator from console results. No automated emulator-launch or FTP module is requested.
4. **Confidence-report QA**: public GitHub issue submission, opt-in consent, old report display, real-console RAM/CPU/revision choices, maintainer review/deduplication and curated-index download. No confirmed hardware entries exist yet.
5. **Batch robustness**: physical interruption, large libraries, cancellation/resumption and incomplete compressed-image identity handling remain limits, not promises. Confirm actual code before describing a planned behavior as shipped.
6. **Security/release hygiene:** unsigned EXE; safeguard secrets, verify GitHub asset hashes and user data survival on upgrades.

The text of [issue #9](https://github.com/TroJanVirus/MagicPatchDesk/issues/9) was created before v0.13/v0.14. Read its later comments and `docs/IMAGE_OUTPUT_WORKFLOW.md` acceptance checklist to decide what needs live QA. Do not blindly reimplement shipped features or close the issue based only on synthetic tests.

## Working procedure for the next agent

1. Fetch `main` and relevant files with the connected GitHub app (or local clone). Locate applicable existing code and tests before suggesting or coding.
2. Check release tag and `core.AppVersion`, open Issues/PRs and latest green CI run. A current chat memory, a previous assistant answer or a dated document is **not** a substitute for fresh repository evidence.
3. Convert the user's new request into a concrete implementation plan against the current code, preserving the above contracts. If an existing user decision answers a question, do not ask it again.
4. Change source in a focused branch; extend regression tests with synthetic assets; `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 cross-build, updater contract; native Windows UI/healthcheck smoke when release changes the GUI.
5. Record truthful QA evidence, open issues, intentional limits and new owner decisions **in the repo**. Do not present published code alone as physical-hardware verification. Update this file and `docs/STATUS.md` after meaningful milestones.
6. Commit/merge the verified work. Publish canonical binary and manifest only after required gates; remove any temporary one-shot publishing workflow after use. Update `AI_HANDOFF.md` and `docs/PROJECT_STATE.md` with dated entries (history) while keeping this file a non-chronological current-state summary.

**If this file disagrees with source code, current releases or a newer confirmed decision, trust the live evidence, document the discrepancy and correct the handoff in the same change.**
