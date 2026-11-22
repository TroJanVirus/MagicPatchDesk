# MagicPatchDesk: instructions for coding agents

## Mandatory new-session bootstrap (read first)

**This repository is the durable context for all future conversations.** Before changing source or answering a project-status question:

1. Read **`PROJECT_CONTEXT.md`** and **`docs/STATUS.md`** for current, evidence-backed implementation, latest verified release, unverified items and the exact resumption workflow.
2. Read **`docs/DECISIONS.md`** for the project owner's explicit requirements, deferrals and immutable constraints.
3. Then read `README.md`, `AI_HANDOFF.md`, `docs/PROJECT_STATE.md`, `docs/UPDATE_CONTRACT.md`, `CREDITS.md` and task-specific documentation/source. `AI_HANDOFF.md` and `docs/PROJECT_STATE.md` are **chronological histories**, not an authoritative current version at their beginning.
4. Fetch current GitHub `main`, the latest release, issues/PRs, and QA results. If source, release or newer instructions disagree with an older document, report and correct the discrepancy. **Never infer the state of the repo from a previous ChatGPT conversation alone.**
5. At the end of every meaningful code/release milestone, update `PROJECT_CONTEXT.md` (current summary), `docs/STATUS.md` (verified vs pending), and `AI_HANDOFF.md`/`docs/PROJECT_STATE.md` (dated evidence) in the same change. Update `docs/DECISIONS.md` only for new explicit owner decisions. Verify that future agents can resume without chat history.

This repository is the authoritative source of the *native Windows* Magic Patch Desk app, not the earlier HTML workbench. This repository is the authoritative source of the *native Windows* Magic Patch Desk app, not the earlier HTML workbench.

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
- Handle XISO/XDVDFS layouts defensively. CCI images are supported by the native CCIM/LZ4 implementation; verify source and split-set compatibility. Never overclaim arbitrary image containers or all xdelta3 streams.
- Keep offline use functional without GitHub or PatchHub; local JMPs in `%APPDATA%\\MagicPatchDesk\\Custom` must work even if sync fails.
- Preserve original patch author credits and links. Do not bundle copyrighted Xbox game images or executables.

## Testing and reporting

For bug reports, record the game title, Title ID, original XBE CRC32, JMP filename, selected groups, and whether the input was `.xbe`, game-partition XISO, or full-disc ISO. Document any unverified hardware, Windows GUI, or game-boot claims explicitly in `docs/PROJECT_STATE.md`.

## Multi-file JMP routing (v0.5)

- `core/targets.go`: a comment such as `#Video (settings.cfg)` is an explicit record target. Normalise relative paths; reject traversal, reserved device names, multiple different candidates and unsafe characters. Never silently apply a named-file record to `default.xbe`.
- `core/xiso_multi.go`: patch existing XDVDFS members by exact relative path or an unambiguous basename only. Full-disc image sector growth is unsafe; refuse it. Preserve original ISO, use staged output.
- `cmd/magicdesk/targeted_windows.go`: for multi-file JMPs, ask the user to locate the named sidecar, skip the named records, or cancel. Preserve originals and store generated external sidecars separately from the selected output. Do not skip without a user decision.
- Converting a multi-file JMP into a single-file format is invalid unless the user explicitly separates the individual target records. Preserve the strict source-file requirements for BPS/Xdelta and search/append JMP conversions.

## v0.6 file-target packs and GUI feedback

Read `docs/PATCH_PACK_FORMAT.md`. A JMP is always one file, even if its records target multiple distinct files. For non-JMP exports containing named targets, create a `.mpdpack` with one standard patch per target and a manifest; never force different files into one IPS/BPS/Xdelta stream. Inspect `notes=` and each record comment for filename.ext references. Per-record wins, one global note target is inherited by otherwise unnamed records, ambiguous notes fail closed. Do not package game binaries or silently skip file targets. Preserve source requirements, SHA256 integrity, and optional JMP CRC32 constraints. Native GDI controls now respond to mouse hover/down/up: do not break pressed-state click dispatch or 16 ms animation timer. Test with `go test ./...`, `go test -race ./core`, `go vet ./...` and cross-build Windows x64.

## 2026-11-21 — v0.7 Rocky5 PF and all-variants pack

See docs/PF_FORMAT.md and docs/PATCH_PACK_V2.md. The PF parser/interpreter is distinct from JMP, and includes CP/HR/OF/SM/SR/MV/RN/RM. Its full-folder application stages a separate tree; ISO file operation changes are refused. New pack schema magicpatchdesk-pack-v2 includes original JMP plus all representable IPS/BPS/Xdelta/EIPS alternatives per named target and source/result checksums; v1 remains supported. Test go test ./..., race, vet and cross-build before releases.

## Diagnostic reporting v0.8

When modifying patch/conversion code, wire failures through `recordFailure` or `recordWarning` in the native UI, and extend `core/diagnostics.go` with stable error categories rather than raw error telemetry. Do not include absolute paths, usernames, game binaries, patch patterns, passwords, arbitrary comments or unfiltered errors in shareable reports. Reports remain local until a user expressly submits a GitHub issue. Tests and Windows cross-build are mandatory.


## v0.9 group IDs

`core/diagnostic_groups.go` computes deterministic IDs from allowlisted error code + operation + validated Title ID. Never include raw error messages, local file paths or game bytes in the group-key inputs or in GitHub issue URLs. Preserve prior JSON reports; recompute missing IDs when grouping. `cmd/magicdesk/diagnostics_windows.go` supplies opt-in GitHub issue search and drafting from the native Settings screen; no background API calls or issue-triggered Actions. Cross-build and run tests after changes.


## v0.10 developer triage dashboard

See `docs/DEVELOPER_DASHBOARD.md`. Keep GitHub API access opt-in and strictly
read-only in `core/developer_dashboard.go`. Avoid telemetry, automatic reporting,
privileged tokens and recurring GitHub Actions. Do not merge groups with
contradictory root-cause references or count issue comments as distinct people.
Maintain safe issue-body marker parsing, bounded pagination, and Windows UI
state locking. Tests are in `core/developer_dashboard_test.go`.

## v0.11: local Recent Files workspace and UI keyboard controls

`core/recent.go` stores the last 18 selected file paths, kinds and UTC usage times in a private `recent-files.json` within the application's data root. It is **not diagnostic data**; do not send full paths to GitHub/issues or include this history in `.mpdpack`. `cmd/magicdesk/recent_windows.go` adds native Recent navigation and keyboard shortcuts. After successful patch writes, conversion writes and PF/pack outputs, call `rememberRecent("output",path)` **outside `app.mu`**, only once files exist. On selecting an output, open its folder instead of automatically treating it as the stock original. Forget/Clear do not delete source files. Keep history writes serialized and check on Windows 10/11. See `docs/RECENT_WORKSPACE.md`.


## Mandatory v0.12 update protocol and maintenance requirements

**Never break the self-updater.** Read `docs/UPDATE_CONTRACT.md` and keep the `--mpd-apply-update-v1` / `--mpd-healthcheck-v1` CLI branches and `MagicPatchDesk-Windows-x64.exe` canonical GitHub Release asset. Every future release must attach `SHA256SUMS.txt`, increment `core.AppVersion`, run `scripts/verify_release_contract.sh` and preserve `%APPDATA%\MagicPatchDesk` data independently of `%LOCALAPPDATA%\Programs\MagicPatchDesk`. The separate updater helper stages, verifies, replaces after exit and keeps `MagicPatchDesk.previous.exe` for recovery. Do not overwrite source files or delete user data.

CCI support is native, CCIM v1/LZ4 with split sets. Read `docs/CCI_SUPPORT.md`; never silently ignore PF/multi-file/pack operations in CCI. Automatic GitHub issue reporting is OFF by default, requires explicit user opt-in and Windows Credential Manager storage, and uploads only sanitized diagnostic summaries; see `docs/AUTO_REPORTS.md`. No automatic GitHub Actions on new reports.


## Mandatory reference attribution and selectable image outputs (2026-11-22)

Every external technical reference, workflow, borrowed idea, specification, asset or codebase used in implementation must appear in **CREDITS.md** with author/project, direct source URL, scope, upstream licence, and reference-only vs actual reuse. Update the app **About** screen (and link to full credits) as credits expand; reference attribution must remain visible to users. Do not copy or link Repackinator's GPL-3.0 source into this independent Go codebase absent explicit licence compliance review, and never imply upstream endorsement.

Read \`docs/IMAGE_OUTPUT_WORKFLOW.md\` before working on game image conversions. The new \`core/game_outputs.go\` has extraction, trimmed XISO writing, CCI output, independent package staging and member verification; \`cmd/magicdesk/game_outputs_windows.go\` adds the native workspace. Respect unsupported source/output combinations; no silent loss of named files or PF operations. Expand synthetic tests and independently verify real Windows builds, source fixtures, CCI splits and physical Xbox bootability before publishing a new release. Keep update helper flags, canonical asset names, privacy controls, and offline operation intact. No automatic GitHub Actions churn.

## v0.14.0-test.3 — patch import and gallery rules

Read `docs/JMP_IMPORT_AND_PATCHHUB_GALLERY.md` before changing patch imports, catalogue metadata or Patch Library artwork. `core.ImportJMP` uses a typed `ErrDuplicatePatch`: exact identical JMP bytes in either Custom or GitHub cache must not be imported or overwrite anything. Same-named but distinct patches require unique filenames, not overwrites. The UI must show **“Duplicate patch found! No need to add.”** in red without a modal dialog, and add a local verification event. PatchHub comparison images are optional and per-patch; do not equate covers with screenshots, assign ambiguous images to a random game/version, invent URLs, or import images into executable assets. Keep asynchronous image downloads safe, bounded, trusted-host-only and compatible with offline caches.

## v0.14.0-test.4 — Mandatory crash and Patch Library safeguards

Read docs/CRASH_RECOVERY_AND_UI.md before touching startup/Win32 callback crash guards, importing JMP/PF, catalogue deduplication, notes, scrolling or Convert & Create fields. The prior diagnostic system only captured handled errors and DID NOT capture process crashes. Maintain durable per-process markers, private stack/stdout evidence where available, next-launch sanitized reports and persistent unreviewed crash notices; never automatically upload raw traces, minidumps or original game data. Treat a forced exit as an abnormal exit, not proof of a specific crash. Preserve updater protocol v1. Do not delete existing duplicate patches: only skip new duplicates and collapse equivalent existing entries for catalogue display. Format JMP notes safely (HTML line breaks etc.) and expose the entire body with scrolling/copy. Separate patch-list and record-list wheel targets; test GUI at ~720p-height. Only show relevant IPS RLE and encrypted IPS password options. Keep all new-session handoff documents aligned.
