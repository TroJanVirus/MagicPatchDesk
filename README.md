# Magic Patch Desk — Native Windows edition

A portable, self-contained Windows x64 application for patching **Original Xbox `.xbe` executables and uncompressed XDVDFS `.iso`/XISO game images**. This is an actual native Win32 GUI — **not HTML, Electron, WebView, .NET, or an installer**. End users only need `MagicPatchDesk-Windows-x64.exe` on Windows 10/11 (64-bit).

## Get started

1. Double-click `MagicPatchDesk-Windows-x64.exe`. The application displays its existing local library immediately while it checks GitHub for the current JMP count and downloads new JMP files into the user cache. It also refreshes a read-only game/title/cover index from Jay's Patch Hub (at most once a day).
2. Search/browse the game catalogue, pick a game and patch, and open an original `.xbe` or `.iso`/XISO. The app can propose a matching patch using the embedded XBE's Title ID and CRC32, but **never applies patches automatically**.
3. Inspect the JMP's commented record groups. Click a group to toggle all its records or click **+** to expand it and toggle individual records. Click **PREVIEW** to dry-run the selected patch, inspecting exact hexadecimal offsets, changed-byte counts and the expected resulting XBE CRC32. No files are written during preview. **APPLY PATCH** also performs a dry run and requests confirmation before writing a separate output copy.
4. In **Convert & Create**, choose an original XBE and modified XBE to create JMP, IPS, Xdelta, or encrypted IPS. Alternatively, select a JMP/IPS/Xdelta in **Patch Library** and choose an original XBE in **Convert & Create** to generate a converted patch. The conversion tab currently expects XBE files, not ISO-to-ISO comparisons.

## Offline use and manual patch packs

- Patch files live under `%APPDATA%\MagicPatchDesk\GitHub` (auto-sync cache) and `%APPDATA%\MagicPatchDesk\Custom` (manually added files). The cache persists between launches.
- No network? Click **Settings → Open patch folder**, place any `.JMP` files or author/game directory trees into `Custom`, and click **Rescan**. You can also drag a JMP or an entire local folder onto the app, or use **Import JMP**.
- To get all game titles and cover art without visiting the website, drag in your PatchHub `titles/` folder or copy it underneath `Custom`. The attached `PatchHub.php` is the PHP site logic, **not the actual `titles` data directory**; it does not by itself contain the entire image library.
- The optional public Patch Hub game index and downloaded cover thumbnails are cached separately so browsing works offline after a successful first sync. These public downloads are not needed to apply patches.
- The app checks `JayYardley/Xbox-Magic-Patches-by-Jay` on startup. It requests the Git tree SHA/count and replaces the local GitHub cache only after downloading and verifying the complete JMP archive. Local `Custom` patches are never deleted by GitHub sync.

## Grouped JMP records and safety

- Recognizes the six headers (`system`, `game-title`, `region`, `version`, `author`, `notes`), CRC32 hashes in curly braces in `version`, search/replace, hexadecimal offsets, and `APPEND:` bytes.
- Multiple records with the **same comment label**, regardless of capitalisation, are shown as one selectable switch. For example, two `#Widescreen` records become **Widescreen ×2**. The `[-]` marker starts an individual record disabled. Click the group to enable/disable all, or expand **+** to set each record independently. Partially enabled groups display a mixed-state marker.
- CRC32 matches the **unpatched XBE**, not the container ISO. If provided, mismatched CRCs are refused. The XBE Title ID is checked when the JMP version contains a parseable Title ID. An optional override is available in Settings and should be used only if the user has verified compatibility.
- A missing search pattern, invalid patch record, or out-of-bounds offset causes an error; failed patches are not silently ignored. **Preview** validates a real patch in memory without touching the original. **Apply** displays a before/after CRC32 and record/byte summary before seeking user confirmation. Full-disc XISO sector-growth constraints are checked during preflight. Output is staged as a temporary file and moved into place only on success.
- Worker count and catalogue category filter are now persisted in `%APPDATA%\MagicPatchDesk\settings.json`. The unsafe CRC32/Title-ID override is **deliberately never persisted**, returning to enforced verification each launch.

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
- The GUI executable has been cross-compiled to a valid PE32+ Windows x64 binary. Core Go tests, including a race-enabled run, passed on Linux. Additional tests cover dry-run validation, per-record/mixed-group behaviour and preferences round-tripping. **GUI interactions and original Xbox hardware compatibility have NOT yet been verified on physical Windows 10/11 or an original Xbox**. Treat as a developer/test build until that QA is complete.

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
