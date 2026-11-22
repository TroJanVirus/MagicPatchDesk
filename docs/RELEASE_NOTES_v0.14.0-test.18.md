# Magic Patch Desk v0.14.0-test.18 — Source-selection crash mitigations and evidence

The user reported further abrupt exits on Windows while repeatedly switching **ISO → CCI → XBE** with a Phantom Crash widescreen JMP selected. The supplied v0.14.0-test.17 crash archive contains one actual background Go runtime panic: `JSON decoder out of sync - data changing underfoot?`, in `core.ReadHubCache` during an automatic Patch Hub metadata refresh. Two other unexpected-exit markers do not contain a usable stack.

## Crash fixes

- **Patch Hub cached JSON no longer kills the patching session through this panic path.** Index reads are bounded and checked; malformed data and decoder panics are recoverable errors. The existing cache remains untouched on failure.
- **Serialised metadata refreshes.** Patch Hub sync and media refresh cannot overlap their writes. Completed cache files are synced and atomically renamed.
- **Background task isolation.** Update-triggered sync and normal Patch Library catalogue refreshes have a recovered-panic boundary. A recovered background panic records a **private local stack** and a sanitized diagnostic notice without terminating the main window or erasing the active crash marker.
- **Win32 pointer-lifetime correction.** The application's custom `sysCall` wrapper now declares `//go:uintptrescapes` so Go-owned UTF-16 strings and Win32 structures remain live across native Windows calls. This addresses a real unsafe-pointer lifetime defect; it is a plausible, **not conclusively proven**, contributor to earlier native access violations.
- **More useful reports.** Crash reports retain up to eight recent source-type transitions (ISO/XISO/CCI/XBE) and a capped selection count, without original game filenames, local paths, patch data, or uploaded private traces. Precise source inspection stages are now retained.

## Regression coverage

The new tests cover deliberately panicking JSON decoders, invalid/oversized cache preservation, concurrent cache reads and writes, recovered background panic notices and stack files, source-type breadcrumbs/privacy, and the mandatory `//go:uintptrescapes` annotation. The pre-existing native Windows smoke cycles **360 actual synthetic ISO/CCI/XBE file selections** with a selected JMP; this release adds garbage-collection pressure to native UI paint passes.

The release pipeline must pass `go test -count=1 ./...`, `go test -race -count=1 ./core`, `go vet ./...`, the Windows x64 build, the real Windows native smoke and `--mpd-healthcheck-v1` before GitHub publishes assets.

## Compatibility and limits

All original game sources are read-only. The selected JMP, record toggles, local report data, and existing caches are retained. The updater remains **protocol v1**, with the unchanged `--mpd-apply-update-v1` and `--mpd-healthcheck-v1` entry points, and assets named exactly `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`.

**The actual user's Phantom Crash ISO/CCI/XBE images were not provided to this build.** Synthetic Windows stress is not proof every particular game image or unrelated native crash is fixed. After updating, select the Phantom Crash JMP, switch ISO → CCI → XBE repeatedly and report any fresh crash evidence from Settings → Problem Reports. Existing issue #19 remains open until field confirmation.
