# Magic Patch Desk v0.14.0-test.6 — JMP branding and PatchHub media

Unsigned developer preview for Windows 10/11 x64. **Existing unresolved v0.14.0-test.5 startup crash issue #19 remains open**: the UI/icon/media changes below are not a diagnosed crash fix.

## User changes
- Sidebar names **Convert Patch** and **Batch Process**.
- Window caption **Magic Patch Desk | Powered by Patch Hub**.
- The supplied transparent **Jay's Magic Patch (JMP)** logo is embedded directly inside the .exe as native Windows icon resources. It is displayed in Explorer, taskbar and app caption without a sidecar image. The Win64 icon resource is built from `assets/jmp-logo.ico.b64` using `scripts/build_windows_icon.py` and validated by native Windows GUI smoke.
- Patch Library now recognises **numbered before/after screenshots** such as the owner's `1. Before.jpg` example, `2. After.png`, `Before1.PNG`, JPG/JPEG/PNG/GIF, and arbitrary image filenames belonging to a specific patch directory. Images whose names don't indicate a comparison phase display **Screenshot**, not a guessed Before/After.
- On selection of one game patch, an asynchronous capped search can discover HTML directory links or probe up to 20 common filename conventions. A **Find images** control permits a retry. Images are verified as actual images before caching to the existing `Covers` directory; downloads obey the 5-MiB bound and trusted HTTPS host. No site-wide brute force or game-data upload.
- The known Phantom Crash image URL provided by the owner seeds the matching USA version-5 patch when that patch appears in the public metadata.
- **Complete arbitrary-filename image discovery needs a published media index on the website.** `scripts/build_patchhub_media_index.py` creates the optional `All-Patches/patchhub-media-v1.json` by enumerating the actual public website image files. The client validates and consumes it if published; no web-host deployment is performed here. Index availability cannot be assumed.
- Source/version/Title ID/region/patch identity checks avoid attaching another version's screenshot.

## Unchanged invariants
The updater still uses `UpdateProtocol = 1`, `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`; no existing original XBE/ISO/JMP, user app data, installer or reports are removed. Release CI must check Go test/race/vet, icon generation and PE resource in Windows x64, native Windows unit tests, the exact EXE healthcheck, new caption/icon GUI smoke and manifest.

## Reliability caveat
The user's Windows Reliability Monitor showed no entry corresponding to two CTDs; an application-private runtime log nevertheless contained a Windows **0xC0000005** access violation while `DispatchMessageW` was called. Without a native faulting module/dump we cannot establish the source; the second crash had no stack. The app now labels a normal event-loop stage `ui_callback` in its private session marker, which makes *future* logs more useful but is **not a fix for the crashes**. [Issue #19](https://github.com/TroJanVirus/MagicPatchDesk/issues/19) remains a release blocker until actual Windows startup is verified.

Full handoff: [PatchHub media and branding details](PATCHHUB_MEDIA_AND_BRANDING.md).
