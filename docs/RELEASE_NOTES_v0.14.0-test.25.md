# Magic Patch Desk v0.14.0-test.25 — cleaner Patch Library, gallery navigation and cover preview

## Changes

- **Remove obsolete cover diagnostics.** Since upstream naming and artwork have been corrected, the Patch Library no longer displays `Cover: ...`, `Check cover`, `Copy details` or `Why no images?`. Its old cover-report generator and forced retry action are removed. The standard `Find images` action remains for manually requesting a fresh comparison gallery; ordinary crash reporting and opt-in diagnostics remain available under Settings.
- **Correct comparison image paging.** The two-up gallery disables **Previous** on the first image pair and **Next** on the last. When there is only one pair, both are disabled. Disabled controls are greyed out, have a normal cursor, and do not navigate or close the viewer if clicked. Existing image ordering and Before/After pairing are preserved.
- **Click a selected patch to unselect it.** The same row now toggles the JMP/PF selection off, clearing its parsed records and restoring **“Choose a patch above to review and toggle its records.”** The selected game, original XBE/ISO/CCI and manually selected output formats remain intact; Apply Patch becomes disabled until a patch is selected again.
- **Visible selected-game cover.** The game detail header gains a fixed 101×100 pixel cover frame, replacing debug controls. Actual covers are letterboxed/pillarboxed to keep original proportions, including square, landscape and portrait artwork. The existing game-list thumbnails are also aspect-fitted.
- **Enlarge on demand.** Hovering the selected cover shows a magnifying-glass badge and interactive cursor. Clicking opens an in-app, large, aspect-fitted cover viewer. Close button, clicking outside, and Esc dismiss the viewer. The patch/record list remains scrollable at typical 1024×720 window dimensions.

## QA and compatibility

A release-stage one-shot workflow must pass full Go tests, Go core race detector, vet, updater compatibility checks, PatchHub media fixtures, official icon generation, Windows x64 PE compilation and SHA256 checks, and actual Windows `--mpd-ui-smoke-v1` plus `--mpd-healthcheck-v1` before publishing.

The native smoke creates an independently generated JPEG cover (no copyrighted game assets), validates the fixed clickable cover and enlarge/close behavior, checks removed cover-diagnostic hit targets, verifies disabled gallery arrows on 2 and 3 images, exercises patch deselection, and retains checks for independent patch and record scrollbars.

No external PatchHub or GitHub JMP files were renamed or modified; no original XBE/ISO/CCI bytes are touched by these UI changes. Self-updater protocol **v1**, executable name `MagicPatchDesk-Windows-x64.exe`, checksum `SHA256SUMS.txt`, portable/installed rollback, saved data and owner-written `README.md` stay unchanged. The real user's manual mouse interactions and mixed image aspect ratios still require confirmation on their Windows machine.
