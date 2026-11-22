# Project state — chronological implementation history

> **Historical development record, not a current-state header.** This file starts with early-build facts from 2026-11-21, then has dated additions through the verified **v0.14.0-test.2** preview. For current QA and active priorities, use [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md), [STATUS.md](STATUS.md) and [DECISIONS.md](DECISIONS.md). Later entries supersede early plans; check live `main` and Releases.

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
