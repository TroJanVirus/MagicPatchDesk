# Magic Patch Desk — Native Windows edition

A portable, self-contained Windows x64 application for patching **Original Xbox `.xbe` executables and uncompressed XDVDFS `.iso`/XISO game images**. This is an actual native Win32 GUI — **not HTML, Electron, WebView, .NET, or an installer**. End users only need `MagicPatchDesk-Windows-x64.exe` on Windows 10/11 (64-bit).

## Get started

1. Double-click `MagicPatchDesk-Windows-x64.exe`. The application displays its existing local library immediately while it checks GitHub for the current JMP count and downloads new JMP files into the user cache. It also refreshes a read-only game/title/cover index from Jay's Patch Hub (at most once a day).
2. Search/browse the game catalogue, pick a game and patch, and open an original `.xbe` or `.iso`/XISO. The app can propose a matching patch using the embedded XBE's Title ID and CRC32, but **never applies patches automatically**.
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
- **Unsupported:** compressed `.cci`/`.chd` or archive formats; Xbox 360 executable patching; arbitrary non-XDVDFS/ISO9660 images; rebuilding the disc layout after arbitrary file growth. Disc images are not re-signed, and modified full-disc images should not be treated as pristine verified disc dumps.

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
- The app only contacts GitHub and Jay's Patch Hub for public catalog/thumbnail updates. It does not upload game files or patch results.
- The GUI executable has been cross-compiled to a valid PE32+ Windows x64 binary. Core Go tests, including a race-enabled run, passed on Linux. **GUI interactions and original Xbox hardware compatibility have NOT yet been verified on physical Windows 10/11 or an original Xbox**. Treat as a developer/test build until that QA is complete.

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
