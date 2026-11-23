# Magic Patch Desk v0.14.0 — first stable release

This is the **stable** Magic Patch Desk release, following the v0.14.0-test.25 preview. The native Windows updater retains its existing protocol **v1**, allowing installed and compatible portable preview builds to update directly to the stable executable with SHA-256 verification and rollback.

## Patch Library fixes

- **Patch-row titles and metadata no longer overlap.** The previous 60px patch cards had a 40px title starting 5px down and another 19px metadata row starting 47px down, causing the metadata to extend beyond its card. The layout now allocates separate text rectangles fully inside each row, aligned beside the actual patch-type badge and preserving useful two-line patch names.
- **No redundant `Find images` button.** Comparison images continue to appear when the automatically maintained Patch Hub index has them; the existing **View images** viewer, Before/After labels and bounded previous/next buttons are unchanged.
- **Simpler notes action.** The button reads **READ FULL NOTES**. The notes remain full-length, formatted and scrollable, with links and group details preserved.

## Cleaner navigation

- **Diagnostics is hidden by default** in the sidebar on new installs. Open **Maintenance → Show Diagnostics tab** to enable it, or choose **Hide Diagnostics tab** to remove it again. The setting is saved in the user's existing `settings.json` and survives restarts. Historical preferences without this field default to hidden. Core crash reports, local evidence, and Settings/Problem Reports remain fully functional even when the developer tab is hidden.
- **Home shows the real Patch Hub logo supplied by the owner**, embedded in the standalone Windows EXE. Its link card reads **Check out the Patch Hub online!** and **So pretty! · jayxbox.com**, rather than suggesting the app's synced catalogue lacks patches.
- **Native titlebar startup refresh.** The top-level Win32 window is first created hidden, then DWM dark/light caption attributes are applied before showing it. After showing, Windows is asked to repaint its non-client caption and border without waiting for a maximize or focus change. The fallback on older supported Windows 10 builds remains.

## Stability, compatibility and QA

The release gate includes full portable Go tests, core race checks, vet, media fixtures, the official embedded JMP icon, EXE PE and SHA256 checks, and execution of **the produced Windows binary** under the native UI smoke and updater healthcheck. The native smoke additionally exercises the supplied Patch Hub PNG via GDI+, ensures the Diagnostics tab toggle and navigation gating, checks patch-row title/metadata bounds and confirms no redundant gallery button, while preserving existing ISO/CCI/XBE stress and patch-result safeguards.

All XBE/ISO/CCI originals remain read-only. GitHub JMP files, Patch Hub media URLs, user-imported artworks, caches, patch record toggles, existing output choices and the user's README are not rewritten by the release. Portability and managed installation use the unchanged `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` filenames; the GitHub Release is **not a prerelease**.

**Manual visual confirmation still encouraged:** titlebar compositing can depend on Windows version and desktop theme. Automated smoke validates the native startup/window API path, not every user's desktop pixel output.
