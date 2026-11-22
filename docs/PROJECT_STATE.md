## 2026-11-22 — v0.14.0-test.10 candidate (README, converter, crash evidence)

- User-supplied end-user `README.md` has been installed **byte-for-byte as parsed text**, without AI rewriting or added boilerplate. All future detailed technical notes go under `docs/` and `PROJECT_CONTEXT.md`, not into the user-focused README. Source numbering changes and About wording are implemented in `cmd/magicdesk/paint_windows.go`.
- Conversion: previously existing source-patch+modified fixture checked original patch and `core.ConvertPatchBytes` independently roundtripped source vs output, but two-XBE creation did not verify the generated patch. Added `core.VerifyPatchAgainstModified` for full-byte output checks across JMP, IPS, BPS, Xdelta and EIPS, called on generated output prior to writing. The worker now snapshots source-patch bytes and file paths rather than reading mutable `app.convertPatchFile`. Also verify optional modified `default.xbe` against generated `.mpdpack` output; when original input is an ISO, extract the true `default.xbe` rather than treat ISO bytes as XBE.
- Unexpected exit v0.14.0-test.9: provided `import_patch` and `ui_callback` crash session markers are separate sessions; neither proves a specific fault. `import_patch` stage previously leaked after the action completed. Restored UI crash stage after completing import, automatic match or patch selection. More importantly, background `reloadCatalog` previously assigned `app.cat`/selection while Win32 paint read it; now publish completed catalogue and auto-match events to UI thread via `WM_APP+2/3`, preventing concurrent replacement of native UI list state while drawing. A private runtime log/WER dump is still necessary for confirmation of actual CTD root cause.
- The patch verification and UI source-order regressions are included; the release is `v0.14.0-test.10` with updater protocol v1 and canonical Windows asset name unchanged.
- Detailed QA limits and user-facing changes: `docs/RELEASE_NOTES_v0.14.0-test.10.md`. Do NOT claim real Windows crash recovery or actual image/hardware testing from Linux Go tests/cross-compile.

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

# Project state — chronological implementation history

> **Historical development record, not a current-state header.** This file starts with early-build facts from 2026-11-21, then has dated additions through the verified **v0.14.0-test.5** preview. For current QA and active priorities, use [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md), [STATUS.md](STATUS.md) and [DECISIONS.md](DECISIONS.md). Later entries supersede early plans; check live `main` and Releases.

Chronology begins 2026-11-21 (NZ); this file retains dated milestones and is not a substitute for current live code/release verification. Automated QA is not a promise of hardware compatibility.

## Implemented

- Go standard-library native Win32 Windows x64 UI with patch browser, conversion, settings, PatchHub catalogue integration and drag-and-drop local imports.
- GitHub JMP catalogue sync on launch, local cache, offline manual `Custom/` library.
- JMP parsing (metadata, CRC32/version, offset, first-match search/replace, append, default-disabled records) and group switches for identical case-insensitive comments.
- IPS parsing and generation; EIPS (custom password-protected IPS); uncompressed VCDIFF subset; stock-vs-patched XBE conversion.
- Original XBE Title ID/CRC32 inspection and patch compatibility checks.
- XDVDFS/XISO loading, default.xbe replacement, game-partition relocation for growth; full-disc growth refusal when beyond allocation.
- Multiworker bounded-memory ISO copying and patch indexing, output staging to preserve originals.

## Tested in this environment

- `go test ./...` passed, including `core` unit tests (2026-11-21).
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -s -w' ...` succeeded.
- A fresh cross-compiled executable matched the prior distributed binary byte-for-byte by SHA-256.

## Not verified

- The GUI has not been executed on real Windows 10 or 11 here.
- Patched commercial games have not been hardware-tested on the original Xbox or xemu in this session.
- Full real-world compatibility across the live JMP repository has not been exhaustively established.
- This EXE is not digitally signed and may trigger Windows reputation warnings.

## Release/repository policy

- Source, tests and documentation belong in Git; binaries belong in Releases.
- Avoid GitHub Actions workflows unless explicitly requested, to minimise CI minutes.
- Preserve patch authorship, backups and user data.
- For contributor context, read `AGENTS.md`, `AI_HANDOFF.md`, and `README.md`.

## 2026-11-21 — v0.5 record filename routing and strict source prerequisites

Implemented in a test branch: filename.ext detection in JMP comment labels, safe per-file selection, multi-file native dialogs (locate / skip / cancel), explicit XISO member resolution and multiple XDVDFS member replacement, source preservation, and a disabled conversion button for input/output combinations requiring originals. Synthetic ISO multi-member and full-disc refusal tests pass. Manual Windows UI and commercial Xbox/XISO boot testing remain outstanding.

## 2026-11-21 v0.6: patch packs and UI hover

Multi-file JMP -> per-target IPS/EIPS/BPS/Xdelta packed into `.mpdpack` (ZIP manifest, per-target verification, no game contents). JMP remains one file. Filename detection now covers `notes=` plus comment, with safe ambiguity resolution. Pack loading/applying handles loose/ISO targets via explicit original file selection. Hover, cursor, pressed states and button animation are implemented in native GDI. Automated Go tests, race checks, vet and Windows x64 compilation pass. Real Windows GUI and Xbox hardware QA are pending.

## 2026-11-21 — v0.7 Rocky5 PF and all-variants pack

See docs/PF_FORMAT.md and docs/PATCH_PACK_V2.md. The PF parser/interpreter is distinct from JMP, and includes CP/HR/OF/SM/SR/MV/RN/RM. Its full-folder application stages a separate tree; ISO file operation changes are refused. New pack schema magicpatchdesk-pack-v2 includes original JMP plus all representable IPS/BPS/Xdelta/EIPS alternatives per named target and source/result checksums; v1 remains supported. Test go test ./..., race, vet and cross-build before releases.

## 2026-11-21 — v0.8 diagnostic reporting

Implemented local classified error and PF warning reports (no raw private paths, passwords, game bytes or patch payloads), Settings report controls and manual GitHub issue handoff. Go tests, race checks, vet and Windows x64 cross-build passed locally. Win32 clipboard/dialog and real-user issue submission still require manual Windows verification. No background telemetry.


## 2026-11-21 — v0.9 local diagnostic groups

Added deterministic group IDs by error type + operation + Title ID, backward-compatible aggregation of old reports, grouped Settings navigation, and user-initiated GitHub issue matching/search before issue creation. Local Go tests, race checks, vet and Windows cross-build pass; real Windows and GitHub UI verification still pending. No recurring or issue-triggered CI.


## 2026-11-21: v0.10 developer diagnostic triage

Native Diagnostics workspace indexes local failure groups and an opt-in
read-only GitHub issue snapshot; prioritizes public report frequency, recency
or open status; links a root issue and a fix PR by explicit body markers;
flags contradictory root-cause references instead of silently merging cases.
Local issue data is never automatically transmitted. Tests and Win32
cross-build passed locally; actual desktop/network testing remains pending.

## 2026-11-21 — v0.11 recent files and keyboard QoL

Added local-only MRU history for game source/patch/conversion inputs/output directories, no automatic submission of paths. New Recent tab with open, folder, copy, forget, clear and pagination. Cross-workspace shortcuts include Ctrl+O/P/F/R, Ctrl+1–6 and F5. All generated outputs are registered after successful writes, never when previewing or failing. Unit tests, race check, vet and Windows compilation passed locally; Windows GUI/Explorer workflows still need manual testing.


## 2026-11-21 — v0.12 maintenance foundation

Added self-install into `%LOCALAPPDATA%\Programs\MagicPatchDesk`, future-release check (stable/preview), canonical update asset + checksum/digest validation, separate helper with explicit v1 CLI protocol and previous-EXE rollback, no user data movement. Added opt-in redacted GitHub Issues upload via Windows Credential Manager and offline retry; auto-report OFF by default. Added pure Go CCIM LZ4 sector reader/writer and split support, CCI↔ISO maintenance tools and default-XBE CCI patch workflow. Windows GUI cross-build, tests, race and vet pass locally; real Windows update flow, GitHub authenticated posting and Xbox hardware CCI validation outstanding. Future release contract documented and checked in CI. No recurring GitHub Actions jobs.


## 2026-11-22: selectable outputs (headless compilation and tests passed)

**New source present:** \`core/game_outputs.go\` and synthetic \`core/game_outputs_test.go\` implement and exercise extraction of XDVDFS files/empty directories, deterministic XISO packing, member-hash verification, CCI conversion, independent XBE/folder/XISO/CCI/full-ISO output choices, no-overwrite stage handling, and prevention of incomplete XBE-only multi-file exports. Windows UI has a **Game Outputs** workspace with independent multi-select formats; PF can stage a complete folder and JMP/pack inputs can target multiple named files. Output choices persist by input category. About credits major external references and links to \`CREDITS.md\`.

**Validated headlessly:** [headless verification run #44913720896](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44913720896) at `c119e657599297f3fa350c8246c3c40bb5b4719c`: `go test ./...` PASS, `go test -race ./core` PASS, `go vet ./...` PASS, and `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` Windows x64 cross-build PASS. **Not yet verified:** Windows 10/11 GUI runtime, xemu, original Xbox and Cerbios physical output compatibility. The GitHub workflow is manual-only after this run. Do not distribute a release as working until all required checks pass. Full-disc image reconstruction from game files is not supported; full-disc ISO output only preserves a supported input layout. Native writer intentionally disallows non-ASCII and invalid XDVDFS names until the required encoding work is verified.

**External provenance:** see \`CREDITS.md\`. Repackinator code is GPL-3.0 and was NOT reused. Its image workflow is referenced; Team Cerbios/libCCI/CCISharp and Rocky5 PF conventions have explicit acknowledgements.


## 2026-11-22: v0.13.0-test.1 preview published

[Preview release](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.13.0-test.1) contains canonical portable Windows x64 EXE and SHA256SUMS for self-updates. Full Linux Go tests, race detection, vet and Windows cross-build, **native Windows Go tests**, candidate digest and `--mpd-healthcheck-v1` execution all passed [run #44914374173](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44914374173). The final GitHub CLI status check used an unsupported field (`prerelease` instead of `isPrerelease`), reporting a CI failure **after** release creation; independent GitHub release API inspection confirmed the release is a prerelease with both uploaded, digest-bearing assets. One-shot publisher workflow deleted. Remaining risk: interactive GUI operation, actual game images / Xbox / Cerbios boots, and install/update rollback require manual verification. NOT declared stable.

## 2026-11-22 v0.13.0-test.2 integrity hardening

Defect reproduced by new Linux unit tests: `discoverCCIParts` panicked on base `.cci` filename while scanning a split set. Fixed bounds check, split-set hole/ambiguity handling. Added XDVDFS 0xFF padding, 64KiB output image alignment and balanced on-disc filename search trees; verifying all directory entries and exact file sizes/hashes now includes previously untested empty folders. Safer Win32 device-name checks, streaming copy truncation detection, case-insensitive named file reading, sidecar-pack title/CRC verification and explicit conversion without patch. Responsive Game Outputs coordinates for 720p desktops and headless `--mpd-ui-smoke-v1` introduced. See `core/game_integrity_test.go`, `core/game_balanced_test.go`, `CREDITS.md` and `docs/RELEASE_NOTES_v0.13.0-test.2.md`.

QA state: synthetic Linux suite, race, static checks, Windows cross-build/native Go tests and a runner-driven UI smoke command are required; hardware/Xbox/Cerbios and interactive Windows 10/11 remain outstanding. One-shot workflow and no recurring GH Actions. Upgrade protocol v1 retained.


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
