# Magic Patch Desk — Native Windows edition

> **New conversation / AI coding agent?** Start at [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md), then [AGENTS.md](AGENTS.md), [current QA status](docs/STATUS.md), and [explicit project decisions](docs/DECISIONS.md). Those files are the canonical, repo-contained handoff; older dated documentation is historical. See the copy/paste first-message prompt in PROJECT_CONTEXT.md.

A portable, self-contained Windows x64 application for patching **Original Xbox `.xbe` executables, XDVDFS ISO/XISO game images and Cerbios CCI images**. This is an actual native Win32 GUI — **not HTML, Electron, WebView, .NET, or an installer**. End users only need `MagicPatchDesk-Windows-x64.exe` on Windows 10/11 (64-bit).

## Get started

1. Double-click `MagicPatchDesk-Windows-x64.exe`. The application displays its existing local library immediately while it checks GitHub for the current JMP count and downloads new JMP files into the user cache. It also refreshes a read-only game/title/cover index from Jay's Patch Hub (at most once a day).
2. Search/browse the game catalogue, pick a game and patch, and open an original `.xbe` or `.iso`/XISO. The app proposes patch matches using the embedded XBE's Title ID and CRC32. Ordinary Patch Library operations require explicit approval; the separate Game Manager supports **user-authorised Auto batch processing** only for unambiguous, CRC-verified games.
3. Inspect the JMP's commented record groups, toggle those you want, and click **APPLY PATCH**. Save to a new file. The input is left unchanged.
4. In **Convert & Create**, choose an original XBE and modified XBE to create JMP, IPS, Xdelta, or encrypted IPS. Alternatively, select a JMP/IPS/Xdelta in **Patch Library** and choose an original XBE in **Convert & Create** to generate a converted patch. The conversion tab currently expects XBE files, not ISO-to-ISO comparisons.

## Offline use and manual patch packs

- Patch files live under `%APPDATA%\MagicPatchDesk\GitHub` (auto-sync cache) and `%APPDATA%\MagicPatchDesk\Custom` (manually added files). The cache persists between launches.
- No network? Click **Settings → Open patch folder**, place any `.JMP` files or author/game directory trees into `Custom`, and click **Rescan**. You can also drag a JMP or an entire local folder onto the app, or use **Import JMP**.
- To get all game titles and cover art without visiting the website, drag in your PatchHub `titles/` folder or copy it underneath `Custom`. The attached `PatchHub.php` is the PHP site logic, **not the actual `titles` data directory**; it does not by itself contain the entire image library.
- The optional public Patch Hub game index and downloaded cover thumbnails are cached separately so browsing works offline after a successful first sync. These public downloads are not needed to apply patches.
- The app checks `JayYardley/Xbox-Magic-Patches-by-Jay` on startup. It requests the Git tree SHA/count and replaces the local GitHub cache only after downloading and verifying the complete JMP archive. Local `Custom` patches are never deleted by GitHub sync.

## Grouped JMP records and safety

- Recognizes the six headers (`system`, `game-title`, `region`, `version`, `author`, `notes`), CRC32 hashes in curly braces in `version`, search/replace, hexadecimal offsets, and `APPEND:` bytes.
- Multiple records with the **same comment label**, regardless of capitalisation, are shown as one selectable switch. For example, two `#Widescreen` records become **Widescreen ×2**. The `[-]` marker starts an individual record disabled; clicking a group applies the new state to every record in it.
- CRC32 matches the **unpatched XBE**, not the container ISO. If provided, mismatched CRCs are refused. The XBE Title ID is checked when the JMP version contains a parseable Title ID. An optional override is available in Settings and should be used only if the user has verified compatibility.
- A missing search pattern, invalid patch record, or out-of-bounds offset causes an error; failed patches are not silently ignored. Output is staged as a temporary file and moved into place only on success.

## ISO/XISO behaviour

- Handles original Xbox XDVDFS filesystems in common **game-partition XISO** and **full-disc ISO** layouts. Finds `default.xbe` in the image, applies the patch to that XBE in memory, and writes an independent ISO image with the updated XDVDFS directory entry.
- For an XBE that grows due to an `APPEND:` record, relocates it to the end of a game-partition-only XISO and updates the directory pointer. For a full-disc/redump style image, refuses growth past allocated sectors rather than writing an unsafe image. Extract/repack the game partition as XISO first in that case.
- Copies large disc images using 1–8 independent I/O workers; also uses workers to index local JMPs. The copy is bounded-memory and gives progress updates. Actual performance varies by disk and controller; multi-worker I/O is not guaranteed to be faster on every drive.
- **CCI is supported:** Cerbios compressed `.cci` images, including split sets, can be decoded, patched and generated through the Game Outputs workflow. **Not supported:** `.chd` or arbitrary archives, Xbox 360 executables, non-XDVDFS filesystems, and unrestricted full-disc rebuilding after file growth. Disc images are not re-signed; modified full-disc images are not pristine verified dumps.

## Patch conversion

| Input/Output | Supported behaviour |
|---|---|
| JMP | Precise offsets, search/replace, append; create new JMP from a stock/patched XBE difference. Generated diffs use offsets and append; they do not infer semantic search patterns. |
| IPS | Standard `PATCH` records, RLE decoding and generation, EOF and final-size support. IPS has a 24-bit addressing limit; oversized changes must use another format. |
| EIPS (encrypted IPS) | Password-protected custom `JMIPSE1!` envelope, PBKDF2-HMAC-SHA256 (210,000 rounds), AES-256-GCM. Compatible with the previous Magic Workbench HTML's EIPS implementation. **Not a standard `.ips` file**: ordinary IPS patchers cannot decrypt `.eips`. |
| Xdelta/VCDIFF | Built-in encoder and decoder for the **uncompressed, default-code-table subset** of VCDIFF. VCDIFF files with custom instruction tables or secondary compression are not supported. Not a full replacement for xdelta3. |

Conversion requires the original file because plain IPS and VCDIFF cannot reliably preserve the source-version identity or original bytes on their own.

## Windows compatibility and trust

- Built with the Go standard library and Windows system DLLs (Win32/GDI/GDI+). No .NET, Chrome, Node, Visual C++ Redistributable, driver, installation, or admin privilege is required.
- **The EXE is currently unsigned.** Windows SmartScreen/antivirus may warn or quarantine it, especially on a new machine. There is no reliable way to guarantee zero warnings without trusted code signing and publisher reputation. Do not disable Windows security to make it run. Consider signing the release using an Authenticode certificate before public distribution.
- The app checks public catalogues, Releases and reviewed compatibility metadata on GitHub and Jay's Patch Hub. Optional issue submission sends only redacted diagnostics or user-consented compatibility evidence to GitHub Issues. It **never uploads game binaries, user game images, raw patch payloads or full local filesystem paths** as reports. Automated diagnostic submission is opt-in and stored credentials remain in Windows Credential Manager.
- The current developer preview additionally passed **native Windows Go tests and executable health/UI paint smoke checks**, as documented in [current QA status](docs/STATUS.md). **Interactive Windows GUI workflows, retail image compatibility and original Xbox game booting have NOT been established on physical test systems.** Treat the release as a developer preview until that QA is complete.

## Build from source (developers only)

Install Go 1.23+ on a developer workstation and run `BUILD-WINDOWS.cmd`, or use:

```cmd
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./core
go build -trimpath -ldflags="-H windowsgui -s -w" -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk
```

Source is under `core` and `cmd/magicdesk`, plus interoperability fixtures in `testdata`. `docs/` may contain later design notes. Keep Jay's patch author credits intact; the GitHub JMP files are **not** bundled with this build, but are synced on launch if reachable.

Feedback and bug reports should include game title, XBE Title ID/CRC32, region, specific JMP filename, and whether the target was loose `.xbe`, XISO or full-disc ISO. Do not distribute copyrighted game images as bug reports.

## Named-file JMP records (v0.5 test)

For a JMP that must patch something other than `default.xbe`, include the relative filename in the record's `#` comment, for example `#720p setting (settings.cfg)` or `#Data table [file=Media/settings.cfg]`. The app detects the filename, offers to locate/use the corresponding file, skip those records, or cancel. In an ISO, a named file must already exist and be uniquely resolvable; in loose-file mode, external assets are saved as new copies under `<output_stem>_patched_files` alongside the output XBE/ISO. Never overwrite original external assets. Full-disc ISO growth beyond allocated sectors is refused. The source XBE CRC32 / Title ID applies to default.xbe only, not to named assets. Multi-file JMPs cannot directly become one IPS/BPS/Xdelta patch; split into one file's records first.

Conversion uses a **disabled Convert button and an explicit reason** when the original is needed but absent. Offset-only JMP → IPS is source-free; find-and-replace/APPEND → IPS requires original; BPS and Xdelta are source-dependent. See `docs/RELEASE_NOTES_v0.5.0-test.md` for limitations.

## v0.6 — Patch packs and responsive controls

A multi-file JMP remains **one `.jmp` file**, but exporting as IPS, BPS, Xdelta or encrypted IPS creates a ZIP-compatible **`.mpdpack`** with one patch for each target file and `manifest.json`. The archive can be opened in Magic Patch Desk or extracted by ordinary ZIP tools. Game content is never included. Details in [`docs/PATCH_PACK_FORMAT.md`](docs/PATCH_PACK_FORMAT.md).

File-target heuristics now inspect **all record comments and `notes=`**. For a note like `Patch driving.xbe`, records lacking a named target inherit `driving.xbe`; per-record filename comments override that. If notes name several files, unnamed records must be clarified to prevent wrong-file writes. URL domains are ignored, and unsafe paths are rejected.

The Convert page asks for the original **of each target** only when necessary, with a Locate target originals button. A multi-file offset-only JMP can become an IPS pack without originals; BPS/Xdelta members or search/APPEND need originals. The Convert action remains disabled until every required original is identified. Input JMPs in JMP output mode remain one JMP. The native UI now has hover/click feedback and changes the cursor over interactive controls.

## v0.7 developer preview: all-formats packs and XBMC4Gamers `.pf`

Select **All Formats Pack** on Convert & Create when converting a JMP. The new v2 `.mpdpack` includes IPS, BPS, Xdelta and (when a password is entered) encrypted IPS for every named target, plus one copy of the original JMP and a manifest. You must supply each original target: the application can read those files directly from a selected XISO. [Pack format](docs/PATCH_PACK_V2.md).

Rocky5 / XBMC4Gamers `.pf` files are a full game-folder operation format, not just a binary delta. Local PFs can be imported and applied through the native Windows app. For a loose extracted game, the tool previews its operations and creates an **independent copy of the game folder** before running any bytes/file changes; ISO application is available only for PF scripts whose operations modify existing file bytes. PF-to-JMP conversion is supported for context-free or source-backed byte edits, and refuses unsupported file deletion/copy/move operations. [PF reference](docs/PF_FORMAT.md).

## Problem reports (v0.8)

Failed patch, preview, conversion, Rocky5 PF and patch-pack operations now create local privacy-restricted diagnostic JSON reports in the application data `Reports` folder. PF warnings are also retained. Settings > Problem Reports lets users view a report, copy its safe summary, open the folder or open a prefilled GitHub issue for the developer. Nothing is uploaded without review and manual submission. Full paths, passwords, game binaries and raw error strings are excluded. See `docs/DIAGNOSTICS.md`.


## v0.9 diagnostic grouping

The Problem Reports section in Settings groups recurring local failures by error category, affected operation, and valid Title ID into stable `MPD-...` IDs. Each report remains separately retained. **Find matches** searches the project GitHub issue titles for an existing group; **Report issue** offers to search first or compose an explicitly reviewed new issue. No reports are automatically uploaded, and the design uses no recurring GitHub Actions. See `docs/DIAGNOSTIC_GROUPING.md`.


## Developer diagnostics dashboard (v0.10)

Open the new **Diagnostics** workspace to combine locally recorded patch
failures with public GitHub diagnostic issues. Click **Refresh GitHub** explicitly
for a read-only snapshot; the app makes no background issue requests. Sort
cases by report count, recency or open reports; browse their linked root cause
issue, fix PR and related failure groups. Copy triage markers or open a
prefilled **unsent** tracking issue in your browser. The original local reports
remain private. `docs/DEVELOPER_DASHBOARD.md` documents the workflow, security
limits, and issue-body markers. GitHub report issue counts are not unique
reporter counts, and neither closed issues nor closed PRs imply a verified fix.

## v0.11 — Recent Files and shortcuts

The Windows sidebar now includes a **Recent** workspace. Reopen previously selected game XBE/ISO inputs and custom patch files, return to conversion sources or inspect output folders, and forget/clear the history without deleting game files. Output files are not automatically used as new originals. Keyboard shortcuts: Ctrl+O (game), Ctrl+P (patch), Ctrl+F (search), Ctrl+R (recent), Ctrl+1–6 (workspaces), F5 (local patch rescan), Esc (leave text entry). The local-only history file is kept separately from all diagnostic reports; full local paths are never uploaded. See `docs/RECENT_WORKSPACE.md`.


## v0.12: Managed install, updates, opt-in automatic diagnostics and CCI

Use **Maintenance → Install app** to place a managed executable at `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe`. The portable EXE can still run independently. Library data remains at `%APPDATA%\MagicPatchDesk`: `Custom/` imported JMP/PF, `GitHub/` synced patches, `Covers/` artwork, `Reports/` local privacy-redacted JSON, `settings.json`, `recent-files.json`, `maintenance.json`. An optional GitHub Issues credential is stored in Windows Credential Manager, never in config files. See [Update contract](docs/UPDATE_CONTRACT.md).

**Maintenance → Check updates / Update now** checks official GitHub Releases, including previews by default, validates the downloaded EXE against the SHA-256 manifest and GitHub digest, restarts using the separate `MagicPatchDeskUpdater.exe` helper and preserves a `MagicPatchDesk.previous.exe` fallback. Versioned releases must always preserve the canonical updater asset and CLI protocol. v0.11 or older portable builds must be upgraded to v0.12 manually once.

**Maintenance → Connect token / Auto-report** (off by default) uploads ONLY privacy-redacted diagnostic JSON to GitHub Issues, grouping repeated failures. Offline errors remain locally queued and can be re-sent with **Sync now**. A GitHub token with issue-write permission is required; users can disconnect it at any time. See [Auto reports](docs/AUTO_REPORTS.md). ChatGPT can then inspect repository Issues through the connected GitHub app, with no need to upload the reports into chat.

**Cerbios `.cci`** is supported with native CCIM v1/LZ4 read/write, split parts, extraction, ISO recompression and direct single-default-XBE patching to a new CCI. More complex PF/multi-file/pack CCI workflows require extraction to XISO first rather than dropping records. See [CCI format and safety](docs/CCI_SUPPORT.md).


## Next development: selectable game outputs and source attribution (2026-11-22; headless checks passed, device QA pending)

The new **Game Outputs** workspace accepts an Original Xbox \`.iso\` (full-disc or game-partition XISO), Cerbios \`.cci\`, standalone \`.xbe\`, or an extracted folder (choose its **root default.xbe**). The chosen input no longer locks the output. Select one or more independent products:

- **Patched original game XBE only** (\`.xbe\`, not an ISO-attacher/stub).
- **Extracted game folder**, including original assets and optional file-specific patches.
- **Rebuilt game-partition XISO** (\`.xiso.iso\` for explicit format distinction).
- **Cerbios CCI** (\`.cci\` or automatic FATX-sized split parts).
- **Layout-preserving full-disc ISO**, only when the input is already a suitable full-disc ISO and its patch can safely fit without expanding its allocated sectors.

Use **Patch Library** to select a JMP/IPS/BPS/Xdelta/EIPS/PF/\`.mpdpack\` if wanted, then choose the source and outputs in **Game Outputs**. Or select no patch and use it as an image extractor/converter. A source \`.xbe\` alone lacks other game files and therefore cannot generate a complete XISO/CCI/folder. A multi-file JMP/pack/PF cannot silently be exported as an XBE-only complete patch; the operation is blocked. PF is processed as a full staged game-folder operation when supported.

Each run saves products inside one new \`<chosen-name>/\` directory and refuses to overwrite existing files. Repack outputs are verified by reopening XDVDFS and comparing the contents; compressed CCI is decoded and checked after writing. Game-partition XISO is **not** a byte-equivalent recreation of a Redump ISO. Intermediate storage must be sufficient for extraction and rebuilt images. The native writer intentionally rejects invalid/ambiguous/case-conflicting XDVDFS filenames and symlinks.

**Headless validation passed:** `go test ./...`, `go test -race ./core`, `go vet ./...`, and Windows x64 `CGO_ENABLED=0` cross-build (Go 1.23.12). Exact run: [headless verification run #44913720896](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44913720896). **Not yet validated:** native Windows 10/11 GUI, real Xbox/Cerbios boot, or a signed/published installer/release. A successful cross-build does not guarantee runtime compatibility.** See [implementation/design details](docs/IMAGE_OUTPUT_WORKFLOW.md), [reference attribution](CREDITS.md), and [work tracker](https://github.com/TroJanVirus/MagicPatchDesk/issues/9).

The **About** screen now lists the reference projects (Repackinator, Team Cerbios/libCCI/CCISharp, Rocky5/XBMC4Gamers, and the JMP catalogue) and links to detailed credits. Repackinator's GPL-3.0 source is not bundled or copied. Future contributions must record exactly which reference work was used and its licensing details in \`CREDITS.md\`.


### v0.13.0-test.1 preview distribution

The [v0.13.0-test.1 preview](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.13.0-test.1) includes the canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`. The release was gated on passing Linux unit/race/vet tests, the existing update-contract check, a cross-built Windows x64 EXE, **native Windows Go tests**, SHA256 validation of the exact distributed EXE on Windows, and executing `--mpd-healthcheck-v1` on Windows. [Full test workflow log](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44914374173). GitHub's release API independently confirms both required assets uploaded, with digests. The one-shot publishing workflow was deleted after publishing; normal pushes trigger no automatic CI. **Interactive Windows UI testing and original Xbox/Cerbios boot validation remain outstanding**, so the release is a prerelease, not stable.

## v0.13.0-test.2 image-integrity follow-up (2026-11-22)

The follow-up preview fixes a reproducible CCI split-name crash; detects missing intermediate CCI parts; verifies all extracted XDVDFS directories, member SHA-256 and exact sizes; emits balanced XDVDFS directory trees with 0xFF entry alignment and 64KiB image padding; rejects Windows-reserved filename aliases; and prevents incomplete case-sensitive folder patch routing. The Game Outputs interface can now **Skip patch** when converting an original image without modifying it, supports `.xiso` in the picker, and adapts to typical 720p Windows desktop height so its export button is usable. Credits were expanded to include Andrew de Quincey and purelogiccode's technical XDVDFS documentation.

A private `--mpd-ui-smoke-v1` CLI paints all Win32 workspaces without network access or writing user files, checks the Game Outputs button visibility in a 1024×720 window and exits. It is **not** an alternative updater protocol. Headless tests are not a substitute for manual Win10/11 or original Xbox/Cerbios boot testing; do not mark the release stable based on those checks alone.

## v0.14 preview — Game Manager, compatibility logs and batch review
**More precise real-hardware reports (v0.14 test 2):** when confirming a patch on an actual Xbox, select the motherboard revision (or **I don't know**) and separately identify whether RAM and CPU are stock, upgraded, or unknown. These self-reported details appear in the public GitHub Issues review and in the individual patch report log. They are not requested for xemu results. Historical reports without details remain readable as **Not recorded**.


**Game Manager** scans an extracted-game/XISO/CCI library without touching the originals, finds exact Title ID/CRC verified JMP matches, and sorts uncertain games for manual review. Select **Auto**, **Review** or **Skip** per game, pick its independent output format and run only the explicitly approved batch. Exact candidates are rehashed before use, and any existing output package is preserved.

The same workspace adds a scrollable **verification log**, read-only **XISO image file browser** with verified single-file extraction, and **hexadecimal byte-overlap inspection** for JMPs. A new, attribution-preserving combined JMP can be exported only with explicit conflict approval and independent source/result verification.

**Field Reports** show individual reviewed **xemu** versus **original Xbox hardware** results with a matching patch SHA256 + game Title ID/CRC32, not percentages. Users may explicitly report working, partial or failed results via GitHub Issues after a consent dialogue; maintainers curate `confidence/reports.v1.json` into the publicly downloadable log. The normal app-update check also checks this public confidence index, preserving the local cache on errors. See [Library Manager and Confidence](docs/LIBRARY_MANAGER_AND_CONFIDENCE.md).

**Remaining testing:** native Windows GUI interaction, real console/game boots, and large retail images. New features are developer preview until verified against representative games.
