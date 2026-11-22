## 2026-11-22 — v0.14.0-test.9 verified preview release

Published: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.9
Verified one-shot run: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44988840325 (success). PR #23 was merged into main. The canonical Windows x64 PE executable `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` were published; digest verification, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, icon build, Windows cross-build and updater contract all passed. Removed the one-shot release workflow afterward, retaining only the manual verification workflow.

The patch ownership and 24h media-cache fixes are released. Owner-provided manifest inspected offline: 477 image URLs in 253 patch directories. The production website's runtime response, Windows GUI screenshot rendering and real Xbox patch applications were NOT independently tested. The current desktop image folder matching is game+patch title (or unique shortened label), not region/revision/TitleID.

## 2026-11-22 — v0.14.0-test.9 image fix candidate

- Owner uploaded the true `patchhub-media-v1.json`: inspected locally, 477 JPG/PNG image URLs under 253 distinct PatchHub patch directories; filenames may be arbitrary. Owner clarified that the **patch folder owns all nested images**; region/revision/TitleID checks should not filter the presentation gallery when the game/patch folder matches.
- Root causes found: `MatchPatchHubMedia` and `AttachHubGallery` reject media by region/version/TitleID; `AppendPatchHubMedia` requires exact HTML revision match; `SyncPatchHub` skips media refresh for 24h when game-list cache fresh. Corrected all four paths, preserving unambiguous shortened patch folder matching, HTTPS image verification and offline cached galleries.
- Release notes: `docs/RELEASE_NOTES_v0.14.0-test.9.md`. v0.14.0-test.9 AppVersion is incremented and updater protocol v1 preserved. Release requires clean Go tests, race detector, vet, native Windows x64 cross-build, release contract and SHA manifest.
- Live website responses, image bytes, native Windows GUI and Xbox hardware cannot yet be claimed validated; the **uploaded file's contents** were examined. Site-only image folders without matching JMP do not create patch payloads.

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

# Current QA and release status — MagicPatchDesk

**Last confirmed:** 2026-11-22 NZ. This is the **current QA status**, not the 2026-11-21 first-build snapshot that previously occupied this file. For the self-contained development handoff and exact next-conversation prompt, see [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md); for user product requirements, see [DECISIONS.md](DECISIONS.md).

## Release-blocking live Windows test failure — 2026-11-22

**v0.14.0-test.5 passed synthetic Linux and native Windows CI, but a user reported TWO CTDs immediately after launch. Do not call the desktop runtime stable.** Private uploaded evidence (`crashes.zip`) contains one native Windows **0xC0000005 access violation** during `DispatchMessageW` (`cmd/magicdesk/main_windows.go:483`) and one additional unexpected exit with no runtime trace; the process marker reports `startup` for both, but this stage has not been updated for ordinary UI dispatch/paint. It is **not** possible to identify the faulting Windows DLL or the precise crashing callback from this Go trace alone. The crash may be in a Win32 callback, GDI/GDI+, external DLL or hook; none is confirmed. The second crash cannot be assumed identical to the first. No game assets or user paths should be posted to public GitHub Issues. **Open regression issue:** https://github.com/TroJanVirus/MagicPatchDesk/issues/19.

**Evidence boundary:** the user checked Windows Reliability Monitor (`perfmon /rel`) and found **no record** at the time. Do not repeat that request. New private app traces or a Windows Event Viewer error (only if one exists) could add a native faulting-module name. Preserve v1 updater and non-overwrite safety. No speculative repaired build should be labelled verified without real Windows startup/field evidence.

## Newly verified in v0.14.0-test.6

- **Full unsigned preview release and native Windows gate passed:** [release v0.14.0-test.6](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.6) and [full gated Actions #44979391498](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44979391498), both Linux and native Windows tests and publisher successful. Canonical Windows EXE SHA256 `639747d63f6deb63add495d352fb8fdd182284a511d8c23979d9c2759434c9a7`; SHA256SUMS.txt attached. Existing v1 updater contract unchanged; one-time publisher removed.
- **User's uploaded JMP logo:** fully embedded Win64 icon via b64 asset `assets/jmp-logo.ico.b64`, Python resource generator and Go linker, visible in native Windows taskbar/window. Windows EXE smoke confirmed real embedded RT_GROUP_ICON/RT_ICON and exact caption **Magic Patch Desk | Powered by Patch Hub**. Sidebar labels **Convert Patch** and **Batch Process** implemented.
- **PatchHub pictures:** accepts the exact owner-supplied `1. Before.jpg` from Phantom Crash USA version5, numbered/uppercase Before and After JPG/PNG/JPEG/GIF, arbitrary patch-specific photo names (labelled Screenshot), safe URL/TitleID/region/revision attribution. Downloads are verified/cached per selected patch. A bounded directory listing /20 HEAD request fallback finds only common filenames; a new optional **media-index JSON** reads all real filenames if the user deploys it on PatchHub. The repository includes `scripts/build_patchhub_media_index.py`. **This index has NOT been deployed to the website** and no complete all-game download was verified: it is not possible to infer arbitrary filenames from one URL alone.
- **Windows CTD remains an open release blocker:** user checked Reliability Monitor and found no corresponding event. Application-private first native 0xC0000005 trace is genuine; second exit no stack. No root cause confirmed, and **test6 is NOT a verified CTD fix**. Crash stage marker now identifies normal UI dispatch as `ui_callback` for future diagnostics. [Issue #19](https://github.com/TroJanVirus/MagicPatchDesk/issues/19) stays open. Real user Windows startup and image rendering must be tested before claiming stability. Full context: [branding and gallery implementation](PATCHHUB_MEDIA_AND_BRANDING.md).

## Latest verified distribution

| Area | Evidence / confidence |
|---|---|
| Released Windows preview | **v0.14.0-test.6** (prerelease, unsigned): https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.6 |
| Published EXE | `MagicPatchDesk-Windows-x64.exe`, SHA-256 `639747d63f6deb63add495d352fb8fdd182284a511d8c23979d9c2759434c9a7` |
| Updater-compatible assets | EXE and `SHA256SUMS.txt` present, validated; updater protocol v1 preserved |
| Latest source-feature PR | **#20:** https://github.com/TroJanVirus/MagicPatchDesk/pull/20 (v0.14.0-test.6, merged after publication) |
| Release QA | **PASS** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44979391498 — Linux, native Windows and publisher |
| Previously validated image and Game Manager previews | v0.13.0-test.2 https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44916327760 ; v0.14.0-test.1 https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44921737286 |
| Curated compatibility reports | **0 reviewed entries** at this snapshot: `confidence/reports.v1.json` revision 1 |

## Passed automated verification (not console testing)

- Linux: `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, updater-contract checks and Windows x64 cross-build.
- Native Windows runner: complete Go tests, release EXE SHA-256 verification, `--mpd-healthcheck-v1` and `--mpd-ui-smoke-v1`, including all five Game Manager tabs and hit-tested Xbox revision/RAM/CPU selection buttons in a 1024×720 window.
- Core synthetic tests include source identity and no-overwrite, CCI split/error handling, game output verification, JMP conflicts and source-relative byte hex windows, curated confidence cache sync and old/new hardware report schemas.
- Automated results **do not** guarantee a visually appealing GUI, correct dialogs under a real Windows desktop session, boots on actual Xbox hardware, or compatibility with each retail release/JMP.

## Newly verified in v0.14.0-test.3

- Full-byte duplicate JMP/PF import protection across Custom and synced GitHub library; harmless red-footer/no-op behavior, separate numbering for differing same-named patches, no overwritten files. Synthetic Go tests passed.
- Safe PatchHub screenshot URL discovery and local per-patch images: before/after pairing, multiple images, conservative Title ID / patch-name association and offline cache metadata. These are validated against **synthetic fixtures**, not an exhaustive live-site screenshot corpus.
- Native Windows EXE smoke now creates a synthetic three-image patch and verifies Patch Library View images, modal navigation/close, and layout at a 720p-height outer window. Linux unit, race and vet; native Windows unit/smoke; EXE SHA verification passed in [run #44971296368](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44971296368).

**Not established:** whether the current public PatchHub PHP page emits screenshot URLs for every patch. The public JMP GitHub corpus does not contain the screenshots. Locally imported PatchHub `titles/` screenshots are supported. Do not fabricate image links or claim complete remote gallery coverage.

## Newly verified in v0.14.0-test.5

- **Code, Windows EXE and publisher gate: PASS.** [Run #44976120121](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44976120121) executed complete Linux Go suite, race detector, vet, Windows x64 build and manifest, native Windows Go tests, checked the exact downloaded release EXE digest and ran `--mpd-healthcheck-v1` and `--mpd-ui-smoke-v1`. The release assets were verified again after publication.
- **Single-game outputs moved into Patch Library:** `CHOOSE OUTPUTS` offers multiple independent selected formats and Preserve current format. Source/patch selection plus explicit valid format is a prerequisite for an enabled Apply Patch hit target; a newly chosen source resets the format mask. Source constraints prohibit fabricating a complete image from a loose XBE and creating full-disc ISO from a non-full-disc input. Real game images and GUI clicks still require field tests.
- **Navigation and Maintenance:** old Game Outputs sidebar entry removed, **Batch Process Library** label replaces Game Manager, Settings/About anchored to sidebar bottom. `Check for Updates` embeds `Update Settings` and clarifies Previews ON. Diagnostic report controls only expose Connect when disconnected and the remaining actions after a Windows Credential Manager connection, with no UI PAT state.
- **Portable update blocker repaired:** test.4 rejected portable update merely because a separately installed app exists, not due to duplicate filenames. test.5 updates the *running executable* in place independently; stages in the running EXE's folder, verifies SHA256 and staged `--mpd-healthcheck-v1`, uses a portable-only PID/target/checksum intent, keeps separate rollback names, and preserves original installed-mode v1 helper arguments. Core tests reject tampered intent, mismatched PID/target/digest and unsafe stage paths.
- **Unverified user environments:** actual portable-in-place self-replacement from a real Windows desktop was NOT performed by CI. An older portable binary with the previous blocker requires a one-time manual download of v0.14.0-test.5; running installed copy may update managed independently. Portable directory must be writable. No physical Xbox boot compatibility claims.
- **Artifacts:** [release v0.14.0-test.5](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.5); published `MagicPatchDesk-Windows-x64.exe` SHA256 `76d1edb3da73cb7a9896c3938820f48802cd4d2237985571411a10545f1c19a4`, with matching `SHA256SUMS.txt`. One-off publisher removed after success; only manual verification workflow remains. See [new UX and updater documentation](PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md).

## Newly verified in v0.14.0-test.4

- **Code and release gate PASS:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44973558561. Go tests on Linux and native Windows; race/vet; Windows x64 EXE build, checksum verification, updater healthcheck and UI smoke of full notes, section-specific scrollbars and conditional IPS controls. Published https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.4. Canonical EXE SHA256 `5cb91f4afe17ca45efe5593b8c004c1f2873fab3455629e37adea3da186ced3b`.
- **Durable failure evidence:** simulated forced exit and caught Go panic produce persistent private crash log, a separate sanitized JSON report, next-launch red unread-crash notice; clean exit and still-running instance do not produce false reports. Native Windows runtime stderr capture is best effort. **Not yet tested:** actual Windows kernel-level crash, original Phantom Crash original-XBE reproduction or manual GitHub issue interaction.
- **JMP duplicate prevention:** semantic comparison beyond raw SHA covers formatting/author/notes/CRLF changes, folder imports and pre-existing duplicate visible catalogue entries; preserves original files and local screenshots. Tests distinguish truly different record bytes/regions, which still import independently.
- **Complete HTML-aware notes and UX:** word-wrap, browser-free `<br>`/list/entity formatting and Copy all notes; explicit scrollable view; cursor-specific patch/record scrollbars with draggable thumbs; IPS password and RLE controls only where relevant.
- **Privacy:** raw stacks and local evidence logs are intentionally private. Sanitized GitHub issue payloads exclude raw failure text/paths and crash evidence IDs. Forced termination and power interruption are classified as abnormal exits, **not** automatically as a confirmed software crash. Windows Error Reporting minidumps are optional via the documented admin-controlled HKLM setup; MagicPatchDesk does not modify those registry policies.

**Open follow-up:** https://github.com/TroJanVirus/MagicPatchDesk/issues/17 records the original Phantom Crash crash reproduction and real-Windows crash log / sanitized reporting acceptance tests.

**Past Phantom Crash cause remains unknown.** We can confirm the previous reporter could not save fatal process crashes, not that any particular duplicate caused the original crash. Read [crash recovery and UI repair details](CRASH_RECOVERY_AND_UI.md).

## Manual QA still outstanding — do not claim completed

1. **Real Windows 10/11 interactive test** of a portable-to-newer-portable update when a separately installed copy exists, with manifest validation, rollback and user-data survival; Patch Library multiple formats/Preserve/disabled Apply/actual output exports; Maintenance conditional connection controls and bottom sidebar, plus reproducing Phantom Crash original-XBE selection (with existing duplicate JMPs), process crash/force-termination then next-launch private crash log/diagnostic upload flow, independent patch/record wheel and thumb hitboxes, notes and IPS controls; also folder picker, drag/drop, DPI scaling, output selectors, report consent dialog and error feedback. Check at small windows and at larger desktop resolutions.
2. **Full library batch test** across legitimate extracted games, XISO/ISO and CCI sources: strict Auto matching, manual queue, Skip, mixed output formats, insufficient disk space, existing destination, interrupted operation, no original modification, proper recovery.
3. **Full-sized image compatibility**, including CCI split files, physical sector/file limits, multi-target named assets, PF and packs, and XBE-only output validation. Not all combinations are supported; fail closed rather than dropping operations.
4. **Actual xemu and physical Original Xbox game boots**. Distinguish emulator versus stock or upgraded Xbox RAM/CPU/revisions; user reports are not official compatibility guarantees.
5. **Field-report round trip:** real Windows GitHub Issues submission with explicit consent, maintainer review of privacy/identity/deduplication, commit to `confidence/reports.v1.json` with revision bump, and live app "new confidence reports" download. Currently no curated reports.
6. **Publisher reputation and signing** remain unresolved; the released EXE is unsigned.

## Known constraints / review-first cases

- **Exact match** auto batch only when Title ID and original XBE CRC agree with a single, verified single-file JMP. Ambiguous games, multi-target behavior and CCI requiring additional XBE identity inspection stay in manual review.
- Full-disc ISO expansion beyond safely allocated sectors is refused; an XBE alone does not provide assets needed to create a complete game image. Supported Xdelta/VCDIFF scope is limited; there is no guarantee of arbitrary compressed streams.
- JMP overlap comparison is a warning, not a proof patches compose semantically. Conflicting writes must be explicitly approved.
- The current library manager is not a fully resumable transactional job engine; long-interruption recovery needs real QA. Always preserve previous game/output files.
- Self-reported hardware details can be explicitly unknown. Older reports without revision/RAM/CPU details say **Not recorded**. xemu reports do not carry physical console choices.

## Active tracking and run policy

- **Open issue #9:** https://github.com/TroJanVirus/MagicPatchDesk/issues/9. Its **original** description predates implemented v0.13/v0.14 features. Later comments describe remaining field QA; do not implement its obsolete "missing feature" list twice or mark its whole checklist verified without evidence.
- Public compatibility submissions first enter GitHub Issues and **must not** silently populate `confidence/reports.v1.json`; curated review is manual / explicitly agent-directed. Exact patch SHA256, original XBE CRC32 and Title ID are mandatory.
- GitHub Actions: only a **manual** verification workflow `.github/workflows/one-shot-verify.yml` remains; no push-triggered release publisher persists after use. Save minutes.
- Before a future release, reconfirm current `core.AppVersion`, latest Release tags and real CI evidence. Any new release needs both `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` with unchanged updater protocol and private-data paths.

**Old chronological status:** earlier dated implementation notes remain in `AI_HANDOFF.md` and `docs/PROJECT_STATE.md`. They are useful archaeology, **not current QA claims**.
