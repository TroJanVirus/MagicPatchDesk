## v0.14.0-test.8 release evidence — 2026-11-22

Preview published successfully at https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.8 (canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` present). The scoped GitHub Actions verification completed **success**: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44985163278. Verified updater protocol, PatchHub media-index generator fixture, native JMP icon embedding, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, native Windows x64 cross-build, PE header, `core.AppVersion`, SHA256SUMS and publication. No new always-on Actions workflow was introduced.

**Still unverified:** hands-on Windows 10/11 light-theme visual contrast and UI clicks, 1024×720 interactive layout, first-launch network sync, actual PatchHub media-index deployment or full website screenshots, and Xbox hardware. Native Windows `--mpd-ui-smoke-v1` assertions were extended but could not be executed on the Linux verification runner. The original web host must serve `patchhub-media-v1.json` for non-enumerable arbitrarily named media.

## 2026-11-22 — v0.14.0-test.8 candidate (new owner request, verification pending)

- Owner reports Light mode has unreadable navy panels/text: redesigned Settings into a dedicated **Appearance** section with four preview swatches and direct persisted theme selection. Introduced a coordinated high-contrast light palette, legacy GDI colour adaptation and window title-bar mode switching. Added 1024×720 Windows UI smoke expectations and theme settings migration tests.
- **Check + sync** bundles app release checking, GitHub JMP and PatchHub website metadata refresh. The single **Check at Start** preference controls all automatic network sync; when disabled, local catalogue still loads and manual controls remain. Combined sync is sequenced to avoid concurrent full catalogue reads.
- Root causes of missing comparison images identified in source: directory derived from GitHub JMP filename, early return on ambiguous XBE revisions before reading manifest, site game cards omit per-patch image filenames, index may be absent, and silent lookup/download errors. New index-first lookup matches game/TitleID/region/known revision/normalised patch identity independent of website folder spelling; bounded filename probing is only fallback. UI now exposes **Why no images?** with errors and logs failures. Genuine cross-region matches remain forbidden.
- Critical upstream-versus-site discrepancy: website AMF Xtreme Bowling **NTSC Widescreen [Esppiral]** media does not have a matching GitHub JMP in the current repository; GitHub has Esppiral **PAL Widescreen**, plus a separate NTSC 720p patch. Do not attach screenshots across patch identities. Arbitrary filenames across all games still require PatchHub host deployment of `All-Patches/patchhub-media-v1.json` generated from server files, as documented in `docs/RELEASE_NOTES_v0.14.0-test.8.md`.
- Candidate v0.14.0-test.8 increments `core.AppVersion`, retains immutable updater protocol v1, canonical `MagicPatchDesk-Windows-x64.exe` + `SHA256SUMS.txt`, portable/managed targets and user data. Await Go tests, race detector, vet, icon generator, release-contract check, Windows x64 cross-build and published release evidence before saying the new preview is available. No physical Xbox or real interactive Windows QA has been performed.

## v0.14.0-test.7 release evidence — 2026-11-22

Preview release published: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.7

Verified successfully by a one-shot GitHub Actions run https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44982148517: updater contract, PatchHub index-generator fixture, Windows icon embedding, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 cross-build, EXE PE signature, SHA256SUMS checksum validation and GitHub preview release upload. The canonical EXE and SHA256SUMS assets are present.

Not yet verified: native interactive Windows testing, a real loaded PAL XBE mismatch, live PatchHub media-index deployment, or physical Xbox game boots. The one-shot workflow is constrained to its own exact workflow file path; do not add continuous CI. The three exact deduplication pairs are available from new Settings -> Duplicates -> View after catalogue refresh; of upstream 803 JMP files, the identical-content GoldenEye Rogue Agent pair has been individually verified.

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

## Latest verified developer preview — v0.14.0-test.6 (JMP icon, public photo discovery)

**Published:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.6 (unsigned preview). **Source PR #20:** https://github.com/TroJanVirus/MagicPatchDesk/pull/20. **Full gated CI:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44979391498 — Linux Go suite, race detector, vet, Windows x64 build, Python web-media index tests, logo icon resource generation, actual Windows Go tests, EXE SHA256, updater v1 healthcheck and Win32 UI smoke verifying the compiled caption/icon **all passed**. Published executable `MagicPatchDesk-Windows-x64.exe` SHA256 `639747d63f6deb63add495d352fb8fdd182284a511d8c23979d9c2759434c9a7`; `SHA256SUMS.txt` included. One-off publisher removed after release to conserve Actions usage. Current `core.AppVersion=v0.14.0-test.6`.

New user decisions implemented: native sidebar **Convert Patch** (not Convert & Create) and **Batch Process** (not Batch Process Library), exact window caption **Magic Patch Desk | Powered by Patch Hub** and uploaded transparent **JMP logo embedded in the EXE** (text b64 source + deterministic `scripts/build_windows_icon.py`; no sidecar file). PatchHub media parser now accepts the owner-supplied Phantom Crash `1. Before.jpg` photo URL, JPG/PNG case/ordinal variations and patch-specific arbitrarily named images. The viewer caches and shows validated photos; per-patch async bounded HEAD/directory discovery does not brute force all games. An **optional complete public media manifest** `/All-Patches/patchhub-media-v1.json` is consumed if provided, generated by `scripts/build_patchhub_media_index.py` run against the **actual PatchHub website file tree**. **The JSON has not been deployed to jayxbox.com**, so all arbitrary image names cannot yet be enumerated on the site. Public game card metadata alone doesn't expose them. Details and host deployment guidance: `docs/PATCHHUB_MEDIA_AND_BRANDING.md`. No bundled game photos or downloaded Xbox game data.

**Crash status not resolved:** user checked `perfmon /rel` for the two test5 CTDs and found **no Windows event**. The app-private 0xC0000005 remains real evidence; no faulting DLL or exact callback known. `main.initApp` source now marks `ui_callback` when entering the message loop to aid future traces. **This is NOT a crash fix**, and no user real desktop startup test of test6 has passed. Issue #19 remains a release blocker. Preserve v1 updater and original user data. Do not automatically reopen a request for perfmon logs already checked; seek new app-private runtime evidence only if a further CTD occurs.

## RELEASE BLOCKER — 2026-11-22: real Windows startup CTDs in v0.14.0-test.5

**User supplied `crashes.zip` with evidence of two unexpected exits immediately after launch. This supersedes any implication that the successful automated release gate proves desktop startup is stable.** Full private trace must not be uploaded to the public repository. Issue: https://github.com/TroJanVirus/MagicPatchDesk/issues/19.

One session (2026-11-22 02:31:42 UTC) contains Windows **0xC0000005 access violation**, PC `0x7ffb695e1548`. Go trace passes through `main.initApp` at `cmd/magicdesk/main_windows.go:483`, but that line calls `DispatchMessageW` in the normal Windows message loop — **not** evidence that initialization specifically caused the exception. The native DLL/module and callback are not named in the Go trace. Second session (02:31:45 UTC) had an empty runtime log, so its exact cause cannot be determined. Both local session markers remained at `startup`, illustrating that current stage tracking does not identify routine Win32 dispatch/paint. A third live session marker was present in the supplied ZIP. No claim is justified that both CTDs share one root cause.

**Next actionable evidence:** user already checked Reliability Monitor and found no error. Only if available, inspect Windows Event Viewer Application Error event or new private app logs; do not request the same Reliability Monitor check again. Go/Win32 synthetic smoke test passed, but has not exercised the user's populated real PatchHub cache or external Windows runtime environment. **Do not publish a speculative "crash fixed" build** without evidence. Preserve updater protocol, user data, and reports. Keep #19 open until actual user startup succeeds.

## v0.14.0-test.5 — unified output and updater improvements (verified developer preview)

**Published:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.5
**Gated release verification:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44976120121 — Linux Go suite, race detector, vet, Windows build, native Windows tests, exact EXE SHA256, v1 healthcheck and 1024×720 UI smoke **all passed**.
**Merged source:** PR #18, https://github.com/TroJanVirus/MagicPatchDesk/pull/18 (squashed after the tagged release).
**EXE SHA256:** `76d1edb3da73cb7a9896c3938820f48802cd4d2237985571411a10545f1c19a4`; canonical Windows EXE and `SHA256SUMS.txt` present, updater protocol **1** preserved.

Six owner-requested changes: remove the redundant Game Outputs sidebar tab and put **CHOOSE OUTPUTS** into Patch Library's game+patch → output flow; **APPLY PATCH is visually disabled and has no hit area** until a patch, game and at least one explicitly selected safe format exist. Choose XBE, extracted folder, game XISO, CCI and (only from real full-disc input) full-disc ISO; allow **multiple independent formats** and **Preserve current format** as an optional source-format choice. Source change resets choices; no implicit remembered format enables Apply. The underlying no-overwrite image engine is retained. Rename Game Manager to **Batch Process Library**. Put Settings and About at the sidebar bottom. In Maintenance rename Self Update to **Check for Updates** and merge Channel & Storage into **Update Settings**, explaining what Previews ON means. When no GitHub credential is connected, show only **Connect** in Automated Diagnostic Reports; reveal Disconnect, Auto-report and Sync Now only after connection.

**Updater blocker identified and removed:** older code deliberately rejected in-app portable updates whenever a separately installed `MagicPatchDesk.exe` existed. This was **not** a downloaded EXE filename collision. v0.14.0-test.5 updates the **currently running EXE** (managed remains managed; portable remains portable), retaining separate backups, validated GitHub SHA256 manifest, new PE healthcheck and an explicit local PID/target/checksum intent for portable helper authorization while keeping the five-argument `--mpd-apply-update-v1` protocol unchanged. A separate installation remains untouched. A user on an **older portable EXE already blocked by this bug must manually download/run the new EXE once**; no release can retroactively rewrite the old updater implementation. The new portable folder must be writable. Regression tests cover both modes but real self-swap on Windows 10/11 remains field QA.

**Current handoff references:** `docs/PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md`, `docs/UPDATE_CONTRACT.md`, `docs/RELEASE_NOTES_v0.14.0-test.5.md`, `core/output_choices.go`, `core/update_intent.go`, `core/output_choices_update_test.go`, `cmd/magicdesk/library_outputs_windows.go`, `cmd/magicdesk/ui_smoke_windows.go`. User-field tests still needed for actual Windows portable update, file dialogs, representative CCI/XISO images, Phantom Crash original-XBE repro (issue #17) and Original Xbox game boots. Green CI is not user/hardware certification.

## v0.14.0-test.4 — crash evidence and reliability repairs (verified developer preview)

**Published:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.4
**Gated release verification:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44973558561 (**all Linux, native Windows and publish jobs passed**).
**Merged source:** https://github.com/TroJanVirus/MagicPatchDesk/pull/16 — merge `8a4dcadc40e92a1a2044c2ff489f633a9b4d4ea6`.
**EXE SHA-256:** `5cb91f4afe17ca45efe5593b8c004c1f2873fab3455629e37adea3da186ced3b`; canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` present, updater protocol v1 unchanged.

The user reported a silent crash when selecting a Phantom Crash XBE after importing a duplicate JMP, along with continued duplicate imports, truncated and literally HTML-formatted JMP notes, irrelevant IPS/EIPS controls and coupled patch/record scrolling. Prior diagnostics could only save handled errors. Implemented durable private process-session markers, best-effort caught-panic stacks/runtime stderr, next-launch **unexpected_exit** diagnostic JSON and private crash log, visible red crash notice and Settings → Problem Reports log-view/review/issue flow. Only sanitized failure category/stage goes to GitHub; raw stacks/dumps never upload automatically. A forcibly killed process/power loss can also be an "unexpected exit" and is not proof of a specific crash. Native Windows system minidumps are not automatically enabled, because documented WER configuration requires administrator-controlled HKLM policy.

Import JMP and folder imports now check conservative **semantic equivalence** (source/game identity, ordered byte changes, groups and targets) beyond file hashes; already-existing duplicate JMPs are collapsed in the visible catalogue without deleting old files. `READ FULL NOTES` displays line-break/HTML-entity aware word-wrapped scrollable instructions and a Copy action. Patch/record lists have separate cursor-targeted scrolling and visible draggable thumb/track bars. Convert & Create shows IPS RLE only for IPS/EIPS/pack, and password only when encrypted IPS input/output needs it.

**Open manual repro / crash-evidence QA tracker:** https://github.com/TroJanVirus/MagicPatchDesk/issues/17.

**Not verified:** the original Phantom Crash failure's precise cause—the previous release captured no stack—and actual clicking/file-dialog behavior on a real user Windows installation, OS-native fatal exception trace availability, commercial Xbox game bootability or full PatchHub gallery coverage. The CI includes synthetic forced exit/panic reporting tests and actual Windows executable UI paint/navigation smoke, not a real Phantom Crash boot test. Consult `docs/CRASH_RECOVERY_AND_UI.md`, `docs/STATUS.md` and the open QA tracker.

## Verified release and repository snapshot

| Item | Status as of 2026-11-22 NZ |
|---|---|
| Primary repository | https://github.com/TroJanVirus/MagicPatchDesk |
| Development source | `main`, native Go and Win32; **not** the older HTML app |
| Latest published release at this snapshot | **v0.14.0-test.6**, an **unsigned developer prerelease**, https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.6 |
| Published canonical Windows executable | `MagicPatchDesk-Windows-x64.exe`; SHA-256 `639747d63f6deb63add495d352fb8fdd182284a511d8c23979d9c2759434c9a7` |
| Required published manifest | `SHA256SUMS.txt` in the same release |
| Latest feature source | PR #20: https://github.com/TroJanVirus/MagicPatchDesk/pull/20 — official JMP icon, renamed navigation and verified PatchHub media discovery |
| Release verification | https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44979391498 — Linux unit/race/vet, Win64 build with icon, native Windows tests, EXE SHA-256, `--mpd-healthcheck-v1`, `--mpd-ui-smoke-v1` all **passed** |
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
| Game-image I/O and independent Patch Library outputs | `core/xiso.go`, `cci.go`, `game_outputs.go`, `output_choices.go`; `cmd/magicdesk/game_outputs_windows.go`, `library_outputs_windows.go`; `docs/PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md` |
| Per-game library scan, batch decisions and cautious execution | `core/library_scan.go`, `batch.go`; `cmd/magicdesk/library_manager_windows.go`, `library_manager_paint_windows.go` |
| JMP overlap, hex visualization, merging, visual log | `core/verification_log.go`, `merge_jmp.go`; Game Manager HEX OVERLAPS / VERIFY LOG views |
| XBE metadata, title/version/CRC32 | `core/xbe.go` |
| Reports and curated confidence events | `core/confidence.go`, `confidence_submit.go`, `report_sync.go`, `diagnostics.go`, `developer_dashboard.go`; `confidence/reports.v1.json` |
| Patch catalogue, metadata and offline cache | `core/catalog.go`, `patchhub.go`, `settings.go`; `README.md` |
| Permanent update protocol | `core/update.go`, `update_intent.go`, `cmd/magicdesk/maintenance_windows.go`, `docs/UPDATE_CONTRACT.md`, `scripts/verify_release_contract.sh` |
| Tests | `go test ./...`, `go test -race ./core`, `go vet ./...`; `core/*_test.go`, `cmd/magicdesk/ui_smoke_windows.go`; `testdata/interop/` |

### Current user-facing workflows

- **Patch Library / Convert & Create:** JMP, IPS, BPS, encrypted IPS, a constrained VCDIFF/Xdelta subset, Rocky5 PF and v1/v2 `.mpdpack` workflows, respecting per-target and source-dependent constraints. Original JMP metadata and author attribution must survive.
- **Patch Library → CHOOSE OUTPUTS:** Select an original `.xbe`, complete extracted folder, XISO/ISO or CCI source and a patch, then explicitly choose any safe combination of output XBE, extracted folder, rebuilt game-partition XISO, CCI or full-disc ISO (only for original full-disc inputs) and/or **Preserve current format**. Apply is inert until a valid choice is made; source changes clear choices. An XBE alone cannot conjure other game assets. Inputs never overwritten; results independently verified. The standalone Game Outputs sidebar tab is gone. Patch-free CCI/ISO conversion remains in Maintenance's dedicated tools.
- **Batch Process Library** (formerly Game Manager): Scan library folders of extracted/XISO/ISO/CCI games. Show per-game patch options with **Auto / Review / Skip**; only one strict Title ID + original-XBE CRC32 verified single-file JMP may be auto-eligible. Recheck patch SHA256 and game identity immediately before writing new packages. Ambiguous matches and CCI images requiring XBE inspection remain for manual review. Complex games should be surfaced after exact matches.
- **Patch Library → Import JMP:** before adding a JMP/PF, hash the full selected file and check both Custom and GitHub libraries. A byte-for-byte duplicate is a **no-op**, with the red bottom-log message "Duplicate patch found! No need to add." and a local verification event; a same-named *different* patch is saved under a new filename, never overwriting an existing patch.
- **Patch Library → View images (N):** where the PatchHub page exposes patch-specific before/after URLs or locally imported PatchHub `titles/` folders contain them, show a native side-by-side comparison gallery with multiple images/pairs, on-demand validated caching and previous/next navigation. Never attach ambiguous screenshots to a guessed patch. The public GitHub JMP corpus contains no PatchHub photos; **live site image URL coverage still needs verification**. Full details in `docs/JMP_IMPORT_AND_PATCHHUB_GALLERY.md`.
- **Batch Process Library → IMAGE FILES:** Read-only image exploration and safe single-member extraction; no forced patch operation.
- **Batch Process Library → HEX OVERLAPS:** Original and competing hexadecimal byte windows. Conflicting JMP writes must receive an explicit warning and manual approval before a new merged JMP; verify resulting content and preserve provenance.
- **Batch Process Library → VERIFY LOG:** Bounded scrollable local human-readable log (steps, errors and pass/warning events), not a large preflight wizard.
- **Batch Process Library → FIELD REPORTS:** Individual dated working/partial/failed entries, **no percent/stars/trust scores**, keyed by patch SHA256 + original XBE Title ID/CRC32. xemu and real Original Xbox testing are **distinct**. Real-console submissions let the reporter choose revision 1.0/1.1/1.2/1.3/1.4/1.6/1.6b/**I don't know**, RAM and CPU each **No upgrade / Upgraded / I don't know**. Historical missing values display **Not recorded**. Physical-console info is never attached to xemu reports. A user must explicitly consent to post a public GitHub Issue; a maintainer reviews and deliberately commits approved reports into `confidence/reports.v1.json`; regular app update-check downloads newly reviewed entries. The index currently has no reviewed confirmations.
- **Maintenance / Settings / Diagnostics:** Check for Updates embeds Update Settings (preview channels, check cadence and folders). The running portable or managed EXE can update independently through the backwards-compatible v1 helper. Automated Diagnostic Reports has Connect only until Windows Credential Manager confirms a GitHub token; Disconnect/Auto-report/Sync appear only afterward. Settings/About stay at the bottom of the sidebar. The existing offline Custom library, private reports and developer issue dashboard remain. Never upload user game files, binaries, paths, secrets or patch contents.

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

0. **Current v0.14.0-test.5 field QA:** reproduce the Phantom Crash original-XBE selection scenario on actual Windows, verify private next-launch crash report/view/upload paths, independent patch vs record scrollbar clicking, full JMP notes and converter controls; also verify the Import JMP duplicate red footer and **real** PatchHub before/after screenshot coverage. Parsing tests and native GUI smoke passed, but website image discovery cannot be certified from HTML fixtures alone.
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
