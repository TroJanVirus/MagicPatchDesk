# Magic Patch Desk

A Windows app for patching original Xbox games. Point it at a game, pick a patch, and it spits out a patched copy. Your original files are never touched.

It works with `.xbe` executables, XISO/XDVDFS disc images, and Cerbios `.cci` images. It's a native Win32 program, so there's no installer, no .NET, no Electron, no browser engine. Just one `.exe` for 64-bit Windows 10/11.

## Getting started

1. Download `MagicPatchDesk-Windows-x64.exe` from [Releases](https://github.com/TroJanVirus/MagicPatchDesk/releases) and run it.
2. On launch it shows your existing patch library straight away, then checks the [patch repo](https://github.com/JayYardley/Xbox-Magic-Patches-by-Jay) in the background and downloads any new patches. It also refreshes the game list and cover art from Jay's Patch Hub (at most once a day).
3. In **Patch Library**, browse or search for a game, pick a patch, and open your original `.xbe` or `.iso`. The app matches patches to your game using the Title ID and CRC32 hash of the XBE.
4. Toggle the individual changes you want inside the patch.
5. Hit **Choose Outputs** and pick one or more formats: XBE, extracted folder, XISO, CCI, or full-disc ISO (where supported). **Preserve current format** keeps whatever you started with.
6. Hit **Apply Patch**. Everything lands in a new, separately named folder. The button stays greyed out until you've chosen a source, a patch and at least one format.

## What it can do

### Apply patches
- Patch loose XBEs, XISOs (game-partition or full-disc), and Cerbios CCIs, including split CCI sets.
- Output as XBE, extracted folder, game-partition XISO, CCI, or a layout-preserving full-disc ISO.
- Each run writes to a fresh output folder and refuses to overwrite anything. Rebuilt images get reopened and checked after writing.

### Convert patches between formats
In **Convert Patch**, give it an original XBE and a modified one and it'll generate a patch for you. It supports:

| Format | Notes |
| --- | --- |
| **JMP** | Jay's patch format. Offsets, search/replace, and append. |
| **IPS** | Includes RLE. Limited to 24-bit addresses, so big changes need another format. |
| **BPS** | Source-dependent, so you need the original file. |
| **Xdelta (VCDIFF)** | Built-in encoder/decoder for the basic uncompressed, default-table subset only. It's not a replacement for xdelta3. |
| **Encrypted IPS (`.eips`)** | Password-protected (PBKDF2 + AES-256-GCM). A custom format, so regular IPS tools can't open it. |

Converting usually needs the original file, since IPS and VCDIFF can't reliably carry that information themselves. If the original is missing, the Convert button stays disabled and tells you why.

### Multi-file patches and packs
Some patches change files other than `default.xbe`. A JMP can name its target file in a record comment, like `#720p setting (settings.cfg)` or `#Data table [file=Media/settings.cfg]`. The app will ask you to locate that file, skip those records, or cancel. Original files are never overwritten; patched copies go in a `<output>_patched_files` folder.

Exporting a multi-file JMP to another format creates an `.mpdpack`, which is just a ZIP with one patch per target file plus a `manifest.json`. **All Formats Pack** bundles IPS, BPS, Xdelta (and encrypted IPS if you enter a password) for every target. No game content is ever included.

### Rocky5 / XBMC4Gamers `.pf` files
`.pf` files are full game-folder operations rather than simple byte patches. You can import and apply them, but the app previews what they'll do and works on a copy of the game folder. For ISOs, only `.pf` scripts that modify existing bytes are supported. Converting `.pf` to JMP works for plain byte edits, and refuses anything involving file deletes, copies or moves.

### Batch Process
**Batch Process** scans a library of extracted games, XISOs and CCIs, finds patches that match by exact Title ID and CRC, and sorts anything uncertain into a manual review pile. You mark each game as **Auto**, **Review** or **Skip**, choose an output format, and run only what you've approved. Originals are never modified.

### Other tools
- **Before/after screenshots:** select a patch and click **View images** to compare side by side. Images are downloaded on demand and cached. Not every patch has screenshots, and the app won't make any up.
- **XISO browser:** look inside an image and extract single files, with verification.
- **Byte-overlap inspector:** see where JMP patches collide. You can export a combined JMP, but only after explicitly approving conflicts.
- **Field Reports:** individual reviewed results for each patch, split between **xemu** and **real Xbox hardware**, matched by patch hash and game Title ID/CRC. These are individual reports, not percentages. For hardware reports you can note your motherboard revision and RAM/CPU setup.
- **Import JMP:** drag in a patch or folder. Exact duplicates (even with different notes or line endings) are detected and not copied. Different revisions with the same filename are saved as numbered copies.
- **Recent:** reopen past games and patches. Keyboard shortcuts: `Ctrl+O` game, `Ctrl+P` patch, `Ctrl+F` search, `Ctrl+R` recent, `Ctrl+1`–`6` switch workspaces, `F5` rescan, `Esc` leave text entry.

## Offline use

The app works fine without internet after a first sync.

- Patches live in `%APPDATA%\MagicPatchDesk\GitHub` (auto-synced) and `%APPDATA%\MagicPatchDesk\Custom` (yours).
- No network? Go to **Settings → Open patch folder**, drop `.JMP` files (or whole author/game folders) into `Custom`, then click **Rescan**. You can also drag and drop, or use **Import JMP**.
- GitHub sync only replaces its own cache after the whole patch archive has downloaded and been verified. It never touches your `Custom` folder.

## How patches are checked

The app is deliberately strict, because a bad patch can mean a broken game.

- **CRC32** is checked against the *unpatched XBE*, not the ISO. If a patch specifies a CRC and it doesn't match, the patch is refused. The Title ID is also checked when the patch has one. There's an override in Settings, but only use it if you've verified compatibility yourself.
- A missing search pattern, a bad record, or an out-of-range offset is a hard error. Nothing fails silently. Output is staged as a temp file and only moved into place on success.
- Records that share a comment label (case-insensitive) show up as one switch. Two `#Widescreen` records appear as **Widescreen ×2**. A `[-]` marker starts a record switched off.

### What happens when a patch makes the XBE bigger
- **XISO (game partition):** the XBE is moved to the end of the image and the directory entry is updated.
- **Full-disc ISO:** the app refuses if growth would run past the allocated sectors. Extract and repack as an XISO first.

## Limits

Not supported: `.chd` or arbitrary archives, Xbox 360 executables, non-XDVDFS filesystems, and rebuilding a full-disc image after a file grows. Disc images are not re-signed, and a rebuilt game-partition XISO is not a byte-for-byte match for a Redump dump.

## Updates

**Maintenance → Install app** puts a managed copy in `%LOCALAPPDATA%\Programs\MagicPatchDesk`. The portable exe also works on its own.

**Check for Updates** pulls from GitHub Releases (previews included by default), verifies the download against a SHA-256 manifest, restarts through a small helper, and keeps a `.previous.exe` as a fallback. Builds older than v0.14.0-test.5 need a manual update.

## Privacy and problem reports

When a patch, or conversion job fails, the app saves a local diagnostic report in `%APPDATA%\MagicPatchDesk\Reports`. Reports leave out full file paths, passwords, game binaries and raw error text. After a hard crash, it recovers the report on the next launch and shows a crash notice. Raw stack traces stay private on your machine.

From **Settings → Problem Reports** you can view a report, copy a safe summary, or open a pre-filled GitHub issue. Nothing is sent without you reviewing and submitting it.

There's also an optional **Auto-report** (off by default). If you connect a GitHub token with issue-write permission, it uploads only the redacted diagnostics, grouped by failure type. The token is stored in Windows Credential Manager, never in a config file. You can disconnect at any time.

The app **never** uploads game binaries, game images, raw patch data or local paths.

Your data is in `%APPDATA%\MagicPatchDesk`: `Custom/`, `GitHub/`, `Covers/`, `Reports/`, `settings.json`, `recent-files.json`, `maintenance.json`.

## Windows warnings

**The exe is not code-signed yet.** SmartScreen or your antivirus may complain, especially on a fresh machine, and I can't promise it won't. Please don't turn off Windows security to get around it. Releases include a `SHA256SUMS.txt` so you can check what you downloaded.

## Status

- Automated tests, race checks, a Windows cross-build, native Windows Go tests and exe health checks all pass (see [docs/STATUS.md](docs/STATUS.md)).
- **Not yet confirmed:** hands-on GUI testing on physical Windows machines, retail image compatibility, and booting patched games on a real Xbox or with Cerbios.
- **Known issue:** there's an unresolved crash on Windows (access violation) tracked in [#19](https://github.com/TroJanVirus/MagicPatchDesk/issues/19). It appears to be tripped after updating within the tool sometimes. Manually update if in doubt.
- Releases are marked as pre-releases until that testing is done.

## Reporting a bug

Please include the game title, XBE Title ID and CRC32, region, the patch filename, and whether you used a loose `.xbe`, an XISO or a full-disc ISO. Don't attach copyrighted game images.

## Building from source

You'll need Go 1.23 or newer. Run `BUILD-WINDOWS.cmd`, or:

```
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./core
go build -trimpath -ldflags="-H windowsgui -s -w" -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk
```

The source is in `core/` and `cmd/magicdesk/`, with test fixtures in `testdata/`. Design notes are in `docs/`. The JMP patch files themselves aren't bundled; they're synced on launch. Please keep the patch authors' credits intact.

If you're using an AI coding assistant on this repo, start with [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md), then [AGENTS.md](AGENTS.md), [docs/STATUS.md](docs/STATUS.md) and [docs/DECISIONS.md](docs/DECISIONS.md). It'll help with what not to mess with, chur.

## Credits

Magic Patch Desk builds on the work of a lot of people: [Repackinator](https://github.com/Team-Resurgent/Repackinator), Team Cerbios (libCCI/CCISharp), Rocky5 (XBMC4Gamers), Andrew de Quincey and purelogiccode (XDVDFS documentation), and everyone who's contributed to the JMP patch catalogue. Repackinator's GPL-3.0 source is not bundled or copied. The full list and licence details are in [CREDITS.md](CREDITS.md).