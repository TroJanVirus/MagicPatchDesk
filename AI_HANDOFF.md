## 2026-11-23 — v0.14.0-test.21 RELEASED and merged; actual Windows cover draw smoke PASSED

- **Release:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.21
- **EXE:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.21/MagicPatchDesk-Windows-x64.exe — 7,862,272 bytes; SHA256 `a774e71b8c1007f6c1b9dd817102b44c303ede7d75b38c556978a9dd9ac9e5e1`
- **Checksum manifest:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.21/SHA256SUMS.txt
- **One-shot release run:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45143154661 — **all three jobs SUCCESS**. Passes media fixture, original updater v1, owner icon, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows PE build/SHA256 verification, real Windows `--mpd-ui-smoke-v1` including verified cached JPEG GDI+ drawing, and `--mpd-healthcheck-v1`. Released only after native Windows checks passed, based on source commit `c2786cd718d3a4cb831a40905e1f020725cfdaa5`.
- **PR #35:** https://github.com/TroJanVirus/MagicPatchDesk/pull/35 squash-merged into main as `a8080c8bba93984e59066ef43a0e5c31d62351d4`. Temporary one-shot release workflow removed from main to conserve Actions minutes.
- **Root-cause investigation:** live diagnostic https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45142481317 successfully retrieved and decoded American Idol (Prototype) image via **actual core downloader** (154,187 bytes) and confirmed exact catalogue URL and `434D0027`. Website/JMP changes not needed. Code flaw was a drop-on-full UI cover scheduler with success-only redraw; test21 replaces with three persistent bounded workers, 48-entry queue, failure/success redraw, per-candidate timeout and invalid-local-cover fallbacks.
- **Still unverified:** user interactive Windows field test of cover appearance and HTTP response from the user's network. Actual automated GDI+ drawing and GitHub-runner HTTP proof do not eliminate other environment-specific faults. Do not claim it definitely fixes every missing cover.
- Keep original README, updater protocol 1, privacy, binary source safety and website/patch filenames unchanged.

## 2026-11-23 — v0.14.0-test.21 one-shot prerelease gate STARTED (not yet verified)

PR #35 fixes image-worker queue starvation, failure-triggered repaint and stale local art fallback after user reports no cover difference in test.20. Live test https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45142481317 confirms website URL, 154,187-byte downloaded JPEG and metadata matching; no website update is needed. Candidate source version raised to `v0.14.0-test.21`. One scoped three-job release gate checks source/updater contract and PE, Go tests/race/vet, **native Windows UI GDI+ JPEG draw smoke**, updater healthcheck, then (only on success) publishes canonical EXE and checksum. No always-on Actions. Do not mark as released until verified GitHub asset and run results. Remove one-shot workflow afterward, preserve human README and live owner field-test requirement.

## 2026-11-23 — Cover delivery regression diagnosed: bounded queue + native GDI+ smoke candidate

After updating to v0.14.0-test.20 and syncing GitHub, the owner still sees the American Idol (Prototype) letter tile. Confirmed the exact JPEG is accessible and the live website shows Title ID `434D0027`. A one-shot live GitHub runner `45142481317` directly exercised `core.DownloadCover` successfully, validating **154,187 bytes**, and `core.SyncPatchHub`/catalogue parsing produced the exact expected canonical image URL and one version. No website or GitHub JMP changes needed.

**Concrete GUI scheduling flaw found:** old test20 `scheduleCoverCandidates` returned immediately when the 3 download slots were full and relied on redraw from *success* only. Failed jobs did not trigger repaint; visible covers could remain unrequested after worker congestion, especially with slow/absent artwork on neighboring games. An invalid local `g.Cover` file could also suppress otherwise-valid fallback art. Candidate fix introduces three persistent workers + 48 bounded requests, mandatory completion-driven repaint, per-URL timeout, short cooldown after failures, and fallback despite stale local art. Added `smokeVerifiedCoverDrawing` invoked by existing real Windows `--mpd-ui-smoke-v1`: creates a tiny verified JPEG in isolated temp cache, resolves via hashed cover path, actually draws via GDI+ and cleans up. This is a candidate only; build, CI and actual user Windows display remain pending.

## 2026-11-23 — v0.14.0-test.20 RELEASED and merged, native Windows smoke PASSED

- **Release:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.20
- **EXE:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.20/MagicPatchDesk-Windows-x64.exe — 7,827,456 bytes; SHA256 `4a6ca99e5e807416e2d61adb571c0fd49008385f1ca35b87d39a4065aec31a35`
- **Checksum manifest:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.20/SHA256SUMS.txt
- **Single gated run:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45141316528 — **all three jobs SUCCESS**: media index fixtures, updater protocol 1, official icon, `go test -count=1 ./...`, race detector, vet, Win64 GUI PE build, checksum verification, actual Windows `--mpd-ui-smoke-v1` and `--mpd-healthcheck-v1`. GitHub prerelease created **only after** native Windows smoke passed, targeting source commit `3d681b23d2bda81fb5c0736b8c36e8f91e1cc201`.
- **PR #34:** https://github.com/TroJanVirus/MagicPatchDesk/pull/34 squash-merged as `af001682a382e64546b5ca643a97b496d485fa84`. Temporary one-shot release workflow removed from `main` to conserve Actions minutes.
- Fix: game-cover fallback via validated, on-page game Title ID and exact PatchHub folder, using bounded JPG/PNG candidates; authoritative card URLs and local files take priority. Existing safe downloader caches for offline use. Limit background cover workers to three without unbounded waiting goroutines. Example American Idol (Prototype) `434D0027`.
- **Unverified:** actual live jayxbox.com download/display on the user's interactive Windows machine and artwork for all missing-cover games. User field test is still required. Do not imply a green synthetic UI smoke proves live artwork retrieval.
- No changed website/JMP filenames or original game binaries; updater protocol v1, the human owner README and all privacy restrictions are retained.

## 2026-11-23 — v0.14.0-test.20 guarded release candidate (one-shot gate initiated)

PR #34 Title-ID game cover fallback changes are staged at `fix/title-id-cover-art-fallback`. Current source `core.AppVersion = v0.14.0-test.20`. Owner requested an EXE test release, so a *one-shot, branch-scoped* release workflow now builds only when its workflow file is first added, preserving GitHub Actions allowance. The pipeline must pass Go tests/race/vet, icon and updater contract, Windows PE build and checksum, and an **actual Windows --mpd-ui-smoke-v1 + --mpd-healthcheck-v1 run**, before it creates a prerelease with `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`. No automatic uploads of diagnostics or user files. No manual change to PatchHub website/JMP or user README.

**PENDING:** Confirm GitHub Actions completion, exact release tag/asset checksums and PR merge before marking released. Real desktop screenshot and live PatchHub download still need owner field test. If gate fails, no test release should be claimed; fix source or checks on a new scoped workflow, avoiding repeated runs. After success remove temporary one-shot workflow from main and update dated handoff with real result.

## 2026-11-23 — Game-cover Title-ID fallback candidate (testing pending)

Owner reports a missing game tile for American Idol (Prototype); website art exists under `All-Patches/titles/American Idol (Prototype)/434D0027.jpg`. The patch lists Title ID `434D0027`. Added display-only cover URL candidates from that known Title ID (and other known IDs up to three) using the exact matching PatchHub folder name, prioritized behind explicit HTML card image URLs and local covers. Only a successfully downloaded, verified image may display; `.jpg` and `.png` are tried, then a 30-minute retry cooldown. Covers use a strict maximum of three running goroutines; fast scrolling cannot queue unlimited workers. No website/JMP names or patch applicability changes.

**Verification:** candidate-builder Go tests passed in isolation; the full repository tests, Win64 build/native UI smoke, live PatchHub URL download and user Windows field test are not yet verified. This is PR source only, not a published release. Keep v0.14.0-test.19 as latest confirmed working release and preserve updater protocol v1, human README and manual-only Actions.

## 2026-11-23 — v0.14.0-test.19 RELEASED and merged; PatchHub region aliases

- **Release:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.19
- **EXE:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.19/MagicPatchDesk-Windows-x64.exe — 7,822,848 bytes; SHA256 `0b72caae6cf58bd0a92b70646178784a1e986ce6d9c7f355ca9e0b95dd2285fe`
- **Manifest:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.19/SHA256SUMS.txt
- **One-shot gated CI:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45017150254 — **both jobs success**. Ubuntu ran updater-v1 check, PatchHub media generator fixture, embedded icon generator, `go test -count=1 ./...` including the exact owner-supplied Bad Boys gallery URL, `go test -race -count=1 ./core`, `go vet ./...`, Win64 GUI cross-build, PE and SHA256 checks. Windows runner passed actual `--mpd-ui-smoke-v1` and `--mpd-healthcheck-v1`, and published EXE and checksum afterward.
- **PR #33:** https://github.com/TroJanVirus/MagicPatchDesk/pull/33 merged into main after release. Release source commit `247927444b8e349faf283bee58d28d11dcaee3f6`. Removed the one-time workflow from main after successful publishing to conserve Actions minutes.
- Bug fixed: region-aware conservative artwork matching when GitHub JMP `(NTSC)` and website image folder `(NTSC+PAL)` share same game/feature/author/Title ID. Existing PatchHub media manifest still supplies URL; no changes to website, repo with original JMPs, executable patch applicability or user data. Both cached gallery and manifest-first selected-patch image lookup share owner resolution with exact priority and duplicate-ambiguity safeguards.
- **Unverified:** actual live `jayxbox.com` image download/display in user's interactive GUI; direct website access was blocked from the development environment. Request Windows field confirmation. The owner already confirmed v0.14.0-test.18 source-switch stability on real game source files.
- Human `README.md` unchanged, updater protocol v1, optional/offline gallery, and read-only input safety unchanged.

## 2026-11-23 — v0.14.0-test.19 regional alias gallery correction (candidate; verification pending)

The prior v0.14.0-test.18 crash regression was confirmed **working as intended** by the owner on their real ISO/CCI/XBE workflow. New issue: GitHub Bad Boys JMP uses `(NTSC)`, whereas published artwork folder uses `(NTSC+PAL)` despite matching full game, Widescreen feature, author Esppiral and XBE Title ID `454D0016`. `patchhub-media-v1.json` already supplies precise URLs; both `AttachHubGallery` and `MatchPatchHubMedia` used exact full label equality and so missed these. Implemented shared `galleryFolderOwners` with precedence exact > semantically compatible explicit region superset > unique abbreviation, conservative author/feature/ID checks and duplicate same-region refusal. Tests cover the real owner URL in manifest parsing, cached attachment, and one-request selected-patch lookup; PAL vs NTSC constraints, different author/feature/TitleID, duplicate source ambiguity, exact priority and paired NTSC/PAL variants. **Not yet compiled, CI-tested, or published until gated workflow completes.** User website and GitHub JMP unchanged, read-only local caches; updater protocol v1, README and assets unchanged.

## 2026-11-22 — v0.14.0-test.18 RELEASED and merged; native Windows smoke PASSED

- **Verified preview release:** https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.18
- **Canonical EXE:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.18/MagicPatchDesk-Windows-x64.exe
- **Checksum manifest:** https://github.com/TroJanVirus/MagicPatchDesk/releases/download/v0.14.0-test.18/SHA256SUMS.txt
- **EXE:** 7,817,216 bytes; SHA256 `b2f223c9224e374b9d141103cf3b511b49e8bbf2d271111e9e8a5e56831b4744`. GitHub Release asset contains this digest and the verified manifest.
- **One-shot gated CI:** https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45008193430 — concluded **success**. Ubuntu: permanent updater contract, media fixture, official icon, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 cross-build, PE and SHA256. Actual Windows runner: `--mpd-ui-smoke-v1` (including 360 synthetic on-disk ISO/CCI/XBE source transitions with selected patch and extra GC paint pressure) **exit 0**, and `--mpd-healthcheck-v1` **exit 0**. Release attached only after tests passed.
- PR https://github.com/TroJanVirus/MagicPatchDesk/pull/32 was merged into `main` after the release; v0.14.0-test.18 source commit `c9cb528dcd69f6818de53d703980d31609154166`.
- Fixes: Patch Hub background JSON decoder panic isolation and checked cache (actual crash trace from user archive), sync serialization, safe background worker reporting, native Windows syscall pointer-lifetime pragma, precise source-switch crash breadcrumbs and regression tests. Keep original README, source-game safety, and updater protocol v1.
- **Field verification still needed:** real Windows 10/11 user interactive Phantom Crash ISO→CCI→XBE cycles, and independent native historical 0xC0000005 root-cause confirmation. A successful synthetic CI suite does NOT certify every actual ISO/CCI/XBE. Keep issue #19 open until field-testing.
- The tag-scoped temporary one-shot release workflow is removed from `main` immediately following success to conserve GitHub Actions minutes; the manual-only verification workflow remains.

## 2026-11-22 — v0.14.0-test.18 release gate initiated (pending verification)

Owner requested publishing the PR #32 crash mitigations. Source version increments to v0.14.0-test.18. A ONE-SHOT release workflow runs only when its own file is first pushed to the exact candidate branch. It gates any GitHub Release on portable Go tests, race, vet, updater contract, PatchHub media fixtures, official icon, Windows PE build, native UI source-switch stress and healthcheck; only then attaches canonical MagicPatchDesk-Windows-x64.exe and SHA256SUMS.txt. **Do not mark this as released or tested until the workflow succeeds and GitHub confirms the release and asset digest.** Remove the one-shot workflow afterward. Keep #19 open pending physical user field test. Human-written README unchanged.

### 2026-11-22 added guard coverage (candidate)

All Patch Library background catalogue reload and user-triggered sync entry points, not just the initial updater check, use panic recovery. Portable AST test checks mandatory sysCall `//go:uintptrescapes` annotation. No app-level tests or user hardware validation executed yet.

### Further 2026-11-22 finding — native sysCall pointer-lifetime defect (candidate correction)

Custom `sysCall(p *syscall.LazyProc, a ...uintptr)` accepted Go-managed Win32 pointers without the `//go:uintptrescapes` compiler annotation that standard `syscall.LazyProc.Call` uses. This unsafe wrapper could leave transient Go-owned UTF-16 and GDI structs unrooted across a native call under GC pressure; actual crashing at that point is unproven. An independent Go 1.23.2 cross-compile probe verified that adding the annotation changes compiler escape analysis to retain the referenced allocation. Applied annotation to wrapper, and forced GC in native UI smoke paint. **No app-level automated Go/Windows test or user field test has been executed for this candidate.** See docs/CRASH_RECOVERY_AND_UI.md and PR #32.

## 2026-11-22 — crash archive root-cause evidence / candidate hardening (not released)

User-supplied `crashs.zip` contains **v0.14.0-test.17** `runtime-217a3d1812a10646d1c2.log` with an actual unhandled Go panic: `JSON decoder out of sync - data changing underfoot?`. Stack points to `encoding/json.Unmarshal` within `core.ReadHubCache` line 99, entered by `RefreshCachedPatchHubMedia` → `SyncPatchHub` → `syncPatchHubMetadata` → `checkForNewRelease` background goroutine. Thus at least one observed CTD was **outside** ISO/CCI/XBE inspection; prior bounded-worker fix alone cannot address this. Go's diagnostic indicates an inconsistent JSON parse, but the archive **does not prove** whether its root cause was a decoder defect, mutated memory, or another unsafe interaction. Two other unexpected-exit markers lack a usable stack.

Candidate fix also captures up to eight privacy-filtered ISO/CCI/XBE source hops and a bounded source-selection count in recovered crash reports (no paths). Bounded and checked optional cached-index decoding (panic converted to explicit cache failure, no overwrite), serialized PatchHub sync/refresh, durable atomic cache writes, independent recovered-background-panic private stacks and sanitized reports, isolated library/metadata maintenance jobs, allowlisted precise game-inspection stage names and synthetic regression tests. Preserve original game assets, self-updater protocol v1 and app offline behavior. **Not yet compiled or field-tested on Windows; no new test release published.** The original Phantom Crash user ISO/CCI/XBE were not uploaded. Do not close issue #19 until actual Windows mixed-source field test and report verification.

## 2026-11-22 — v0.14.0-test.17 released; real Windows 360-switch stress passed

Release: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.17
One-shot CI: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44999014598 — **completed SUCCESS**, including:
- `go test -count=1 ./...` with 120 repeated actual ISO/CCI/XBE inspections, byte-identical source checks and context cancellation cases; `go test -race -count=1 ./core`; `go vet ./...`; updater protocol-v1 contract and media-index tests; native Windows x64 cross-build, PE verification and SHA256SUMS validation.
- **Actual Windows executable `--mpd-ui-smoke-v1` exit code 0 on Windows runner**, exercising 360 rapid actual on-disk ISO/CCI/XBE switches with a selected synthetic patch. The smoke validated final source wins, patch group/record toggle states unchanged, **`inspectorMaximum<=1`**, idle inspector, bounded memory and Windows file handle counts, as well as all prior UI smoke assertions.
- Windows `--mpd-healthcheck-v1` updater health check passed.
- Published exact canonical `MagicPatchDesk-Windows-x64.exe` (**7,788,544 bytes**, SHA256 `1185454a98e612b3833353c5d3819dec76c275fc3a562dd87c3b026acd4720d3`) plus `SHA256SUMS.txt`. No ZIP required.
- The confirmed code defect was **unbounded concurrent background inspections**: old `openInput` spawned one independent goroutine every time a file was chosen, while stale-result checks did not cancel parsing/decompression/memory retention. v0.14.0-test.17 now uses one worker plus one replaceable pending job, cooperatively stops old work during ISO/CCI/XBE reads and reduces per-part CCI index peak memory by eliminating the duplicate raw index buffer. App shutdown cancels the pending/running job.
- Human owner-written `README.md` unchanged, all old v1 updater entry points/assets and source read-only guarantees intact. The original user's exact **Phantom Crash ISO/CCI/XBE** was NOT provided for test. Therefore this **is not proof that a particular real image can never trigger a separate crash**, nor proof of the historical Windows exception address; user should field-test repeated selection on actual files. Source stress reproduction + Windows synthetic stress provides strong structural verification.
- Release notes: `docs/RELEASE_NOTES_v0.14.0-test.17.md`. A one-shot CI workflow was used and should be removed from `main` after merging so it cannot consume future Actions minutes.

## 2026-11-22 — v0.14.0-test.17 candidate: repeated mixed-source crash overload

- Owner clarified an actual reproducible workflow: **Phantom Crash patch selected → Open file ISO → CCI → XBE → repeat**; application exits on repeated rapid switches, including CCI, despite v0.14.0-test.15/16 fixing previous single-selection race and synchronous ISO classification.
- **Concrete source flaw:** `openInput` launched a *new goroutine on every selection*. The existing `gameInputGeneration` checked only the result, not the ongoing scan; previous XDVDFS parsing, CCI full-sector indexing, large XBE reads, decompression and open handles continued concurrently. This was an unbounded workload with resource pressure and plausible native exit cause (exact exception still unconfirmed without WER dump).
- New `cmd/magicdesk/game_inspect_windows.go` implements ONE worker (`inspectorOnce`), one queued **latest-only** replacement, context cancellation of previous active/queued work, UI-thread-only selection and output-format commits with retained generation/path check, and cancellation on Win32 `WM_DESTROY`. Captures HWND at scheduling to avoid worker reading mutable UI state. Internal atomic active/max/completed/cancelled counters allow a real native stress smoke assertion. No longer spawn one goroutine per switch.
- `core/source_inspection.go` provides cross-platform `InspectGameSourceContext` for XBE, ISO/XISO, split ISO and Cerbios CCI. `core/xiso.go` now uses `OpenXISOContext`, `openXDVDFSReaderAtContext` and `Image.ReadFileContext`; existing APIs delegate to `context.Background`. Directory traversal checks cancellation, and embedded file reads are chunked up to 1 MiB with cancellation checks. `core/cci.go` has `OpenCCIContext` index parsing with cancellation; `core/cci_inspect.go` has `ReadAtContext`, `ReadCCIGameMembersContext` and `InspectCCIExecutableContext` so multi-part sector decoding can be interrupted between sectors.
- CCI index peak memory reduced: rather than retain the entire large raw byte index *and* a decoded uint32 index concurrently, stream index decoding 8192 entries/32KiB at a time into one retained decoded index. Keep all CCIM integrity validations.
- New portable `core/source_inspection_stress_test.go` covers 120 ISO/CCI/XBE inspections preserving exact TitleID/whole-file CRC and original image bytes, plus cancellation. New `cmd/magicdesk/rapid_source_smoke_windows.go` runs 360 actual rapid image changes in a Windows executable, with a preselected synthetic JMP, latest-only selection checks, record toggles preserved, max concurrent inspectors <=1, and bounded handles/memory. Hooked into the native `--mpd-ui-smoke-v1` suite.
- **NOT YET VERIFIED** until CI run: all portable tests/race, Windows compilation/actual native Win32 smoke, and updater contract. User actual Phantom Crash ISO/CCI files not provided; do not claim exact retail image field compatibility or native fault root cause proven. Owner-written `README.md` unchanged, updater protocol 1 and canonical assets stable. Release notes: `docs/RELEASE_NOTES_v0.14.0-test.17.md`.

## 2026-11-22 — v0.14.0-test.16 published; native Windows smoke verified

Preview: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.16
Scoped release workflow: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44997796855 — **all three jobs completed successfully**:
- Ubuntu portable `go test -count=1 ./...` covering actual built XISO fixture (game partition/full-disc), 7,500-entry skewed XDVDFS tree, truncated members, metadata mutations, unexpected short reads; `go test -race -count=1 ./core`; `go vet ./...`; native Windows x64 PE cross-build and SHA-256 manifest verification, updater v1 contract, official icon and media fixture.
- **For the first time in these recent releases, an actual Windows GitHub runner executed `MagicPatchDesk-Windows-x64.exe --mpd-ui-smoke-v1` AND `--mpd-healthcheck-v1`, both EXIT 0.** Native smoke exercises simulated ISO-pending / validated-full-disc / xiso / invalid / stale-source UI conditions, existing selected-JMP state preservation, plus the wider existing UI smoke suite. This is stronger evidence than Windows compilation alone; it is NOT the same as user field testing on a Phantom Crash retail ISO with an interactive desktop.
- Gated publishing of `MagicPatchDesk-Windows-x64.exe`, **7,757,824 bytes SHA256 `196b9b0c50bc0d90d112f545cf7c478c656aab30c5f3b5312231743abeb09918`**, and `SHA256SUMS.txt` completed success.
- An earlier CI run uncovered an existing synthetic two-file XISO fixture that overlapped XBE and settings data; the new identity test was corrected to build an actual XDVDFS via `BuildGameXISO`. Next run caught two leftover variable references in Win32 source classification; removed them. No build was released from either failed gate. The final third scoped run was successful. Avoid future CI churn by checking both portable test fixtures and Windows-only refactor compilation before kicking off the release job when a local cross-build is available.
- Original owner-written `README.md` untouched, updater protocol `1` and canonical release assets unchanged. **Root cause of historical CTD still unproven** because prior crash evidence lacks a WER exception or full private runtime trace; owner said test.15 seemed stable and needs to field-test the actual Phantom Crash ISO in test.16. Never claim a complete real-file compatibility proof without those bytes/test.

- Audit follow-up: extracted game folders also await root `default.xbe` verification before outputs unlock; Game Manager manual review distinguishes folders and inspects their root XBE, not the directory as an ISO.
## 2026-11-22 — v0.14.0-test.16 candidate: precise ISO picker audit and additional hardening

- Owner clarified reliably reproducible older Windows exit: **Phantom Crash → select library JMP → Open file → choose ISO → immediate CTD**, not a loose XBE. They report `v0.14.0-test.15` no longer crashes and request a thorough ISO-focused audit. Source ISO itself was **not uploaded**; prior abnormal-exit recovery marker `v0.14.0-test.13` has no runtime crash stack or WER exception pointer. Do not claim confirmed root cause or actual real-ISO reproduction.
- Found further real UI-thread problem after test.15: `selectGameSource()` still synchronously called `gameSourceCategory()` → `core.OpenXISO()` to classify ISO as full-disc vs game-partition. That runs the complete directory scan **inside the native file-picker callback**, potentially blocking/reentering the UI. Now classify only through worker `inspectGamePath`: return real sourceKind (`xiso`, `fulliso`, `cci`, `xbe`) along with identity in `gameInspection`, and publish via the preexisting `WM_APP+4` event. No image scan is performed on the UI thread just for classification.
- New `gameOutputValidated` source gate means `currentGameOutputKind` and `availableGameOutputs` return unavailable until an image has passed XDVDFS and `default.xbe` inspection. **Do not infer full-disc safety from filename.** `setGameOutputValidation` checks selected source path and supports an extracted folder's root `default.xbe`. Stale results cannot re-enable output choices for an unrelated source. Game Manager's manual review now invokes `openInput` to use same asynchronous validation instead of bypassing safety.
- `core/xiso.go` hardened: `readAt` rejects short nil-error reads and invalid ranges; regular-file requirement for ISO; named member bounds revalidated before reading; XDVDFS binary-tree parsing uses an explicit iterative pending stack instead of recursive nodes; total metadata bytes limited to 128 MiB and entry count to 200,000. Existing subdirectory nesting limits preserved. These limits prevent large malformed trees causing memory/stack exhaustion.
- New portable tests in `core/xiso_picker_regression_test.go` for valid game-partition and full-disc layouts, exact original XBE identity, 7,500-node skewed directory tree, altered/truncated files, invalid root table, short reads and 100 header/directory mutations without panic. New native smoke `smokeISOPickerValidation` for pending/validated full-disc vs xiso outputs, stale and invalid scans; existing native output smoke updated for verification prerequisite.
- Release notes: `docs/RELEASE_NOTES_v0.14.0-test.16.md`. Version v0.14.0-test.16, updater protocol 1, original owner-provided README and canonical release assets unchanged.
- **Testing still pending**: Go tests/race/vet, Windows x64 cross-build and SHA manifest, and especially native Windows smoke/real Phantom Crash ISO field test. Release must not claim real Windows stability was proven by Linux CI.

## 2026-11-22 — v0.14.0-test.15 Windows stability preview published

Release: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.15
Gated scoped CI: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44996305193 — **completed success**. Verified updater protocol v1 contract, PatchHub media-index fixture, official icon, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 native GUI cross-build, PE signature and SHA256SUMS checksum. Published exact canonical portable EXE `MagicPatchDesk-Windows-x64.exe` 7,751,680 bytes SHA256 `0f5cc5f820cbf790e14a977c3ef7de7e67e507a2f17461bd965c064512e7b3ed`; published `SHA256SUMS.txt`. User-provided human README unchanged and persistent updater contract intact.

**Still unverified:** real Windows `--mpd-ui-smoke-v1` execution, physical Windows interaction selecting the original XBE after a chosen JMP, and the actual fault/instruction pointer for the user's v0.14.0-test.13 exit. Their uploaded recovery marker is NOT a runtime trace; `runtime-8ace874d26c39493f96a.log` or WER minidump is needed to confirm root cause. The implementation is a grounded mitigation of a real data race and duplicate re-selection, **not a proven crash fix**. This caveat must be preserved in future release claims.

## 2026-11-22 — v0.14.0-test.15 candidate: crash on original XBE selection after choosing JMP

- Owner reported an abrupt exit on choosing a known-correct XBE after a library patch was already selected. Supplied **v0.14.0-test.13** evidence: previous-session `crash-8ace874d26c39493f96a.log` only says last stage `ui_callback`; separate new `session-1d856e7d7833149fe585.json` is another process; `failure-f7fca5b403f0136752.json` is an automatically created `process.unexpected_exit.ui_callback` diagnostic, **not an exception stack**. We have NOT seen `runtime-8ace874d26c39493f96a.log` nor any WER crash dump. Do not claim root cause proven.
- Code inspection found a real background-to-UI data race in `openInput`: the worker assigned `app.inputInfo`, `app.inputPartSummary` and `app.status` while Win32 `drawLibrary` and `drawGameDetails` read the XBE's CRC / Title ID. The old `openInput` reset the marker to `ui_callback` before worker inspection returned, making such exits poorly attributed.
- New `cmd/magicdesk/game_inspect_windows.go` isolates file reading from UI model changes. The worker publishes an immutable `gameInspection` result via a buffered channel and `WM_APP+4`; `commitGameInspection` validates **both** the game input generation and absolute selected path before updating UI on its owning thread. Multiple rapid file choices cannot commit stale results.
- `autoSelectXBE` preserves a patch deliberately chosen for that game, including its in-memory group/record toggles, rather than rerunning `selectPatch` and another media-discovery request. It reports a nonmatching CRC as a warning without silently changing the patch; no-match discovery likewise retains explicit selection. Original mismatch checks remain inside patch application.
- Loose XBE inspection now checks a regular file, minimum header length and 512 MiB maximum before read. Crash stages distinguish `game_source_open`, `game_inspect_pending`, `game_inspect_commit`, `auto_match`, with generic stage on completion. Native Windows smoke `smokePreselectedPatchThenXBE` validates selected JMP preservation and rejects obsolete worker result, but Linux CI can only compile the smoke.
- Human README unchanged, updater protocol v1 and canonical assets unchanged. New documentation `docs/RELEASE_NOTES_v0.14.0-test.15.md`. Release build/test and hands-on Windows retest pending; original CTD cause remains **unconfirmed** until runtime panic log or Windows Error Reporting dump is reviewed.

## 2026-11-22 — v0.14.0-test.14 published and verified

Release: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.14
Successful one-shot run: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44995671424 — all required gates completed **success**: permanent updater v1 contract, media-index fixture, JMP Windows icon, `go test -count=1 ./...` (new HTML/URL safety and full comment tests), `go test -race -count=1 ./core`, `go vet ./...`, native Windows x64 GUI cross-build, PE identification and SHA256 manifest verification.

Canonical asset `MagicPatchDesk-Windows-x64.exe`: **7,742,976 bytes**, SHA256 `e0c2f70ebaa593cef6bd5a53074512ba8603ac1abe4c96a46c7a4169ab3aa6a7`. `SHA256SUMS.txt` published. Early scoped CI attempts caught and fixed Go's RE2 regex repeat-count ceiling and a quoted test fixture syntax issue; the final successful run includes both fixes. `README.md` stays untouched, updater protocol `1` and canonical release assets retained. No claim of actual Windows GUI smoke execution, real third-party link clicks, physical Xbox testing or prior crash resolution. Changes are presentation-only; notes and JMP group/comments remain stored verbatim and patch toggle behavior is unchanged.

## 2026-11-22 — v0.14.0-test.14 candidate: complete record group comments + useful links

- Owner screenshot shows **Phantom Crash** JMP group title cut off: `Widescreen Support (Everything is less stretched ...` despite available space. Some group comments and `notes=` headers include vital HTML, full URLs, instructions and multi-line explanatory content. Requirement: make them readable/useful without altering patch semantics or losing anything. Human owner `README.md` remains untouched.
- Actual clipping source in `drawGameDetails`: `textAt(hdc, strLimit(g.Name, 50), ...)` for each group, a deliberate **50-character cutoff**. Replaced with a **two-line wrapped preview**, larger row height (60), and distinct **Read** button per group. Reading group details is intentionally separate from its existing checkbox/toggle and + record expansion.
- New modal uses the existing Win32 scrollable notes UI for either `notes=` or full `Group.Name` selected via `notesTargetGroup` (-1 means header notes). Link navigator `Previous / Open link / Next / Copy URL` is shown only for valid HTTP(S) URLs. URL launch requires explicit confirmation, validated with `core.SafeJMPLinkURL` before `ShellExecute`. No HTML execution, WebView or remote content.
- `core.FormatJMPNotes` preserves HTML `<a href>` link destinations visibly in text while converting HTML paragraph/list/break tags and entities to plain text; `core.ExtractJMPNoteLinks` parses anchor, Markdown and bare HTTP(S) URL references with deduplication, safety bounds and rejection of script/file schemes, credentials and control characters.
- UI provides **Copy original** for complete unaltered HTML or long raw comments (viewing is still bounded to avoid huge Win32 GDI allocations). Where space permits, also provide **Copy text** for readable form.
- New portable tests `core/jmp_notes_links_test.go` and Windows smoke assertions cover long comments, valid/invalid links, grouping unchanged, toggle vs Read hitboxes, and header-notes switching.
- Release version is `v0.14.0-test.14`, updater protocol v1, canonical assets and patch data untouched. Native Windows GUI execution still unverified if Linux cross-build only; previous crash remains open. More: `docs/RELEASE_NOTES_v0.14.0-test.14.md`.

## 2026-11-22 — v0.14.0-test.13 published successfully

Preview: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.13
One-shot CI: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44994029736 — **completed success**. `scripts/verify_release_contract.sh`, PatchHub image index fixture, native JMP icon resource, `go test -count=1 ./...` including new synthetic multipart CCI and numbered raw ISO/XISO cross-boundary/member/output tests, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 GUI cross-build, PE check and `SHA256SUMS.txt` checksum verification all passed.

Canonical Windows EXE `MagicPatchDesk-Windows-x64.exe` (7,727,104 bytes, SHA256 `82da5c57e4f5d33320f2cbe2a41bb35df96c3d4524e7c63bf202d62236a31fda`) and `SHA256SUMS.txt` are published. The owner-supplied `README.md` remains unchanged. Updater protocol `1`, exact asset naming and rollback contract are preserved.

**Still unverified:** native Windows `--mpd-ui-smoke-v1` execution, live Win32 clicking/dragging and rendering, retail split-image compatibility, physical Cerbios/Xbox bootability and elimination of prior unexpected exits. Linux Go unit tests / Windows cross-build do NOT establish those claims. Unknown absent trailing parts cannot always be detected without an external manifest, and arbitrary split naming formats are not supported. Source-safety and test details: `docs/RELEASE_NOTES_v0.14.0-test.13.md`.

## 2026-11-22 — v0.14.0-test.13 candidate: Home, drag-and-drop, multipart images

- User requests: (1) complete multi-part image awareness, including correct embedded `default.xbe` extraction regardless of which numbered part is supplied, (2) dropping ISO/XISO/CCI must identify the game and jump into Patch Library, like loose XBE, (3) a friendly Home landing page with brief features, prominent drag/drop guidance and a direct Patch Hub link. Owner-provided `README.md` must remain the human-oriented file, unchanged unless justified.
- Existing `core.OpenCCI` already resolves `.1.cci` / `.2.cci` / ... contiguous sets with a 64-part safety bound. But `openInput` only validated the CCI header and set `app.inputInfo` empty, so it could not auto-match the game. Added `core/cci_inspect.go` implementing `io.ReaderAt` for the decoded logical sector stream, a shared XDVDFS parser that accepts `io.ReaderAt`, and `InspectCCIExecutable` to extract only the `default.xbe` data and CRC/Title ID. Multi-target `gameOriginals` now uses one batch read from CCI instead of decompressing a full temp ISO.
- Numbered raw `.1.iso`, `.2.iso` or `.1.xiso`, `.2.xiso` sets also now use a read-only virtual `core/split_image.go` reader for lookup, member extraction and safe game-partition outputs. `ExtractGameXISO` uses the combined ReaderAt. Cannot copy/compress just one selected part or incorrectly preserve it as a full-disc image; split raw sources are classified as safely re-exportable game XISOs, not untouched Redump ISO.
- All part readers validate first/middle part availability, duplicates, regular files, CCI/XDVDFS metadata and logical member spans. The selected part and total are shown in Patch Library; dropping many siblings at once is treated as one game selection. Missing unknown final parts without references are not conclusively detectable from a directory listing alone; do not promise otherwise.
- All dropped `.xbe`, `.iso`, `.xiso`, `.cci` switch to tab 0 (Patch Library) and inspect game metadata asynchronously; `autoSelectXBE` reuses Title ID / CRC / game name ranking, suggests a verified patch without auto-applying, and shows a clear no-match message if needed.
- New native Home tab ID **9** is the default landing screen, preserving every older tab ID; it includes native Open game / Browse patches actions, support explanations and an external Patch Hub link `https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php`. Sidebar compacts at short heights to avoid covering bottom Settings/About or the status footer; smoke checks cover Home navigation. No webview or remote data is embedded.
- Regression tests `core/cci_inspect_test.go`, `core/split_image_test.go` exercise synthetic multi-part CCI and raw ISO/XISO: cross-part default.xbe / named files, XBE CRC32, selection of later parts, missing middle/first part failures, safe extraction/independent outputs and unsafe partial-copy refusal.
- Release notes: `docs/RELEASE_NOTES_v0.14.0-test.13.md`; task docs `docs/CCI_SUPPORT.md`, `docs/IMAGE_OUTPUT_WORKFLOW.md` updated. `core.AppVersion` bumped to v0.14.0-test.13 and updater protocol v1/canonical Windows asset names unchanged. **Tests, actual GUI interaction, physical Cerbios/Xbox boot and any prior crash resolution must not be claimed before verification.** README kept untouched.

## 2026-11-22 — v0.14.0-test.12 verified preview release

Preview published: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.12. Single gated workflow https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44992546561 **completed success**: release contract, PatchHub media fixture, Windows icon, `go test -count=1 ./...` (new source-free JMP→IPS/EIPS, source-replace and BPS constraint tests), `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 GUI cross-build, PE format, SHA256 manifest. Release asset `MagicPatchDesk-Windows-x64.exe` 7,698,944 bytes SHA256 `fd86b66dfba51c713eecfee1a480c53d5e120566e066b65e00b27e520c2a63ff`; canonical manifest `SHA256SUMS.txt` present. Native Win32 `--mpd-ui-smoke-v1` assertions compile with the binary but were **not** executed on Windows; physical GUI/hardware and unrelated crash resolution remain unverified. Source-free BPS is explicitly NOT supported because source/output CRC and source length are missing.

## 2026-11-22 — v0.14.0-test.12 candidate: direct JMP→IPS and BPS requirements

- Owner: “converting jmp (from library or imported) to bps or ips does not require original xbe for context if the jmp patch is offset only ... block conversion if jmp DOES contain search and replace functions and state that this is the reason why.” **Technical distinction:** IPS can be generated without the XBE from absolute edits; BPS cannot be valid without source length/source CRC32 and target CRC32. Do not emit a misleading broken BPS. Make that unavoidable limitation explicit in UI and release notes. User's core goal is source-free use when genuinely possible.
- Root cause of apparent source requirement: selecting an imported JMP forcibly changed `app.format` to `core.AllPatchFormats`, which requires originals for BPS and Xdelta. Now default to IPS for selected library JMP, newly picked JMP and drag-dropped JMP; preserve the user's explicitly chosen output format through a `convertFormatChosen` flag.
- `core.PlanPatchConversion` now analyzes enabled JMP records before format-specific checks and gives explicit search-and-replace (or APPEND) source-free block reasons for IPS/EIPS; explains the real BPS source-size/source-and-target-CRC requirement even on offset-only JMPs. With a valid original, search-and-replace conversions remain supported; without one they are hard-blocked. Keep IPS EOF/range safety constraints.
- `refreshConversionHint` now propagates `plan.Reason` instead of replacing it with generic source required; blocked action exposes full reason in a native dialog, and footer can wrap. Multi-file JMP pack planning similarly surfaces search-and-replace and BPS/Xdelta source requirements per target.
- Added portable regression tests covering direct IPS/EIPS, no-original errors for search-replace, BPS source dependence and output equality, plus a Win32 smoke assertion for default library JMP→IPS and format preservation. Release notes: `docs/RELEASE_NOTES_v0.14.0-test.12.md`.
- `README.md` remains the owner's unchanged human-readable file. AppVersion v0.14.0-test.12, updater protocol v1, canonical release asset names and patch/saves untouched. Automated release gate and native UI/hardware QA status to be updated separately.

## 2026-11-22 — v0.14.0-test.11 published and verified

Preview release: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.11
Successful one-shot workflow: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44991571756. Windows x64 canonical `MagicPatchDesk-Windows-x64.exe` (7,692,288 bytes, SHA256 `8124c591fde95c922977ab138705a81d923f73e14f239ad89d88634c41eeab93`) and `SHA256SUMS.txt` published. Verified updater protocol v1, image index fixture, embedded native JMP icon, `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, Windows x64 GUI cross-build, PE signature and SHA manifest. Actual Win32 `--mpd-ui-smoke-v1` assertions for text fit at 1024px/800px and real visual rendering have **not** run on a Windows desktop in this Linux CI environment. Owner must field-check screenshots/text and can use release v0.14.0-test.10 as rollback. This release does not claim to resolve prior unrelated crashes. User-facing `README.md` unchanged.

## 2026-11-22 — v0.14.0-test.11 candidate: UI legibility at real desktop widths

- Owner supplied 3 Windows screenshots showing `Widescreen` patch badges and long game names clipped, `Discover _patch` / `Updates _maintenance` headings (Windows treats `&` as an accelerator), Maintenance `CHECK + SYNC AT S...`, `App fol...`, `Data fol...` labels, and an unhelpful `OUTPUTS: Choose a source game` prompt. Request is to make buttons and relevant descriptions fully readable.
- Changed native GDI `txtRaw` to always pass `DT_NOPREFIX` so literal ampersands render correctly. Added real `GetTextExtentPoint32W` font measurement and compact caption fonts to the shared `button` renderer; when a label still does not fit, a sufficiently tall button uses a wrapped caption instead of preemptively truncating.
- Updated Maintenance layout to give `App folder`, `Data folder`, and `Check + sync at start` adequate width; at narrower widths, buttons reflow into two rows and the update panel becomes taller. Added 1024px and 800px Win32 smoke bounds/text-fit checks; note that smoke tests compile but do not execute in Linux Actions.
- Adjusted Patch Library sidebar width dynamically on larger windows, wrapped game titles and patch names, measured patch category badge widths, and improved header text. Rephrased the output prompt to `Choose the formats to create from your selected game and patch`, with wrapping in the footer, retaining explicit source/patch/output safety gates.
- Owner-supplied human `README.md` remains **untouched**. No game data or conversion behavior changed. AppVersion is `v0.14.0-test.11`, preserving updater protocol v1 and canonical Windows asset names. Release notes: `docs/RELEASE_NOTES_v0.14.0-test.11.md`.
- Await Go tests/race/vet, Windows cross-build and SHA-manifest verification. Native Windows visual rendering and crash absence are **not** established by Linux CI.

## v0.14.0-test.10 release evidence — 2026-11-22

Published preview: https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.10. Canonical 64-bit Windows PE asset `MagicPatchDesk-Windows-x64.exe` (7,683,072 bytes, SHA256 `992ec908323efb259b45064df8cd2deb0d8cb2577269bd75e90061920947d7f4`) and `SHA256SUMS.txt` present. Verified single scoped workflow completed **success**: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44990975380, including updater v1 contract, media generator tests, native icon generation, `go test -count=1 ./...` (new JMP/IPS/BPS/Xdelta/EIPS round-trip cases passed), `go test -race -count=1 ./core`, `go vet ./...`, native Windows x64 cross-build and SHA checksum validation. Earlier run failed **only** because a workflow step assumed `go test -c` emits an EXE for a package with no `_test.go` files; removed that invalid check, no code/test fix was needed. Windows native smoke compiled but could not be run on Linux CI. No real Windows GUI or Xbox hardware operation has been confirmed. Crash cause remains unknown without private runtime log or WER minidump.

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
