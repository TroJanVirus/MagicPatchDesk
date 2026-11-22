# Current QA and release status — MagicPatchDesk

**Last confirmed:** 2026-11-22 NZ. This is the **current QA status**, not the 2026-11-21 first-build snapshot that previously occupied this file. For the self-contained development handoff and exact next-conversation prompt, see [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md); for user product requirements, see [DECISIONS.md](DECISIONS.md).

## Latest verified distribution

| Area | Evidence / confidence |
|---|---|
| Released Windows preview | **v0.14.0-test.3** (prerelease, unsigned): https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.3 |
| Published EXE | `MagicPatchDesk-Windows-x64.exe`, SHA-256 `9ddc49f889b86ffca10ef13e5feaa8b0bac351493970ce43d71dd6d02374ea60` |
| Updater-compatible assets | EXE and `SHA256SUMS.txt` present, validated; updater protocol v1 preserved |
| Latest source-feature PR | #15: https://github.com/TroJanVirus/MagicPatchDesk/pull/15 (v0.14.0-test.3; merged after release publication) |
| Release QA | **PASS** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44971296368 |
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

## Pending verification — v0.14.0-test.4 reliability work

The current feature branch adds durable abnormal-exit report recovery and private crash logs, stronger duplicate JMP recognition across imports and pre-existing library entries, HTML-aware full notes, separate Patch Library scrollbars and conditional IPS/EIPS controls. Go and Windows smoke regression tests were added, including simulated forced-exit recovery, stack privacy and two independently scrollable lists. **These changes are not yet a verified public release**; use current GitHub Actions and release evidence before updating this status. The past Phantom Crash failure cannot be attributed to a specific panic or corrupt JMP without a crash trace. See docs/CRASH_RECOVERY_AND_UI.md.

## Manual QA still outstanding — do not claim completed

1. **Real Windows 10/11 interactive test** of game folder picker, drag/drop, patch controls, wheel/scroll handling, DPI scaling, output selectors, report consent dialog and error feedback. Check at small windows and at larger desktop resolutions.
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
