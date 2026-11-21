# Magic Patch Desk v0.13.0-test.2 — Image integrity and desktop reliability

**Testing prerelease — not certified on physical Xbox hardware or through interactive Windows 10/11 GUI use.**

This release hardens the v0.13 Game Outputs workspace after an expanded failure-mode audit:

- Fixes a **panic** in Cerbios CCI split detection when a standalone \`.cci\` file appears alongside numbered CCI segments.
- Rejects **missing middle CCI segments**, split-set gaps, duplicate/ambiguous CCI selections, and unexpected non-regular part files rather than silently operating on a truncated set.
- Writes **balanced XDVDFS directory lookup trees**, instead of a long right-only list, so folders with hundreds or thousands of files avoid deep traversal.
- Adds XDVDFS **0xFF directory-entry padding** and rounds images to **64KiB boundaries**, following published format descriptions.
- Validates **empty and nested directory preservation**, file sizes and SHA-256 round-trips. Rejects case-colliding files, unsafe reserved Windows device names (including multiple suffixes), symlinks, and truncated copy operations.
- Handles **case-insensitive named-file lookups in extracted folders** and verifies source XBE CRC/Title ID even for patch packs that modify sidecars only.
- Adds **Skip patch / Include patch** in Game Outputs so users can always make clean format conversions without clearing Patch Library selections.
- Adds \`.xiso\` to the input picker and scales the Game Outputs interface for smaller Windows desktops, keeping the export button away from the status bar.
- New optional \`--mpd-ui-smoke-v1\` test-only CLI paints the native workspaces and checks accessible Game Outputs controls at 1024×720 without syncing patches or modifying game files.
- Explicitly credits [Andrew de Quincey’s Xbox DVD filesystem documentation](https://multimedia.cx/xdvdfs.html) and [purelogiccode/XISOStudio](https://github.com/purelogiccode/XISOStudio) in [CREDITS.md](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/CREDITS.md). These sources informed format verification only; their implementation code was not copied.

## Tests and remaining limitations

The release job must pass Go tests, race detector, static analysis, Win64 GUI compilation, native Windows file-system tests, and headless Windows app health/UI smoke checks **before** publishing. SHA256SUMS.txt and the canonical updater filename are mandatory.

**Manual QA still required:** GUI interaction on actual Windows 10 and 11 devices, extracted-folder and CCI/XISO game startup on xemu and original Xbox/Cerbios, FATX USB/disk write behaviour, and installed self-update/rollback under a live user session. These tests cannot be truthfully inferred from CI. This remains a **preview**, not a stable or hardware-certified build.

To upgrade from v0.12 or v0.13 test.1, use **Maintenance → Check updates** with preview releases enabled. Your existing app data, patch folders, diagnostic reports, and rollback EXE are intentionally preserved by the unchanged v1 self-update contract. You can also obtain the canonical portable Windows EXE from this prerelease and check its SHA-256.
