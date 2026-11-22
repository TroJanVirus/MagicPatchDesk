# Magic Patch Desk v0.14.0-test.3 — Duplicate-safe JMP import and PatchHub screenshots

Developer **prerelease** (unsigned Windows x64 executable). Nothing here certifies a patch or game has been tested on physical Xbox hardware.

## Changes

- **Import JMP:** checks the full patch bytes against previously imported Custom patches and synced GitHub patches. An exact duplicate is **not** copied, and the bottom status line shows **"Duplicate patch found! No need to add."** in red. Also logged locally under Game Manager → VERIFY LOG. No modal popup or unnecessary rescan for the duplicate.
- **Safe file collision handling:** two differently authored/revised JMPs with the same name retain both file copies under uniquely numbered filenames. Original files are never overwritten. Invalid/symlink inputs fail closed.
- **PatchHub multi-image gallery:** optional **View images (N)** on the selected patch in Patch Library. Browse several before/after screenshots, two at a time, fitted to their original aspect ratios, with next/previous/close navigation.
- **Artwork sources:** patch-specific screenshot links published within PatchHub game-card HTML/metadata, plus screenshot files in imported per-patch PatchHub `titles/` folders. Trusted-host, image size and content checks are reused; remote photos are downloaded only when viewed and retained for offline use.
- **Backwards-compatible cache:** older PatchHub metadata caches still load. The first metadata check after installing this version requests a gallery-aware refresh, then resumes the normal daily cache window. Game cover thumbnails are unchanged.

## Limits and privacy

The public GitHub JMP patch repository contains no PatchHub comparison screenshots. If a particular PatchHub game card doesn't publish the image URLs in its HTML, they cannot be recovered by guessing filenames; those patches simply don't display a gallery. Locally imported PatchHub screenshot folders are supported. Real website gallery coverage and actual photo rendering still need manual validation.

No source image, XBE, ISO, original JMP, user path or token is sent to a third party by importing. All downloaded images remain in `%APPDATA%\MagicPatchDesk\Covers`. No change to updater v1, stable `MagicPatchDesk-Windows-x64.exe` / `SHA256SUMS.txt` release asset contract, or installation folders.

## Automated verification

Release publication is gated on Linux Go tests + race/vet, native Windows build and tests, exact EXE SHA256 verification and `--mpd-healthcheck-v1` / `--mpd-ui-smoke-v1` including gallery navigation with synthetic screenshot filenames. Actual Windows 10/11 mouse/file-dialog testing and a representative live PatchHub gallery still require field QA.

Full technical details: [JMP import and PatchHub gallery](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/docs/JMP_IMPORT_AND_PATCHHUB_GALLERY.md).
