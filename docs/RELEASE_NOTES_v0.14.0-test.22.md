# Magic Patch Desk v0.14.0-test.22 — cover diagnosis (NOT a confirmed fix)

**Why:** The owner reports that cover art still does not display after v0.14.0-test.20 and .21. We will not pretend another speculative download retry solves it. This build exposes the actual missing evidence from the user's Windows environment.

## New controls (Patch Library → select a game)

- **Cover status:** Shows a concise local state (no candidates, downloading/queued, HTTP 404/403, TLS/DNS/network timeout, decoded cache valid, or GDI+ load/draw error).
- **Check cover:** Force a bounded, asynchronous retry for the selected game's known website URLs, without changing source files, JMP records or other cover caches. Run once, let it complete.
- **Copy details:** Copies a *privacy-filtered* per-game report for the user to paste into ChatGPT. It includes version, public game Title IDs, exact published candidate URLs, last error class per URL and cache validity. It never includes username, local directories, file bytes or raw network errors.

## Evidence so far

GitHub one-shot live probe https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45142481317 successfully downloaded and decoded the actual **154,187-byte** American Idol (Prototype) JPEG and retrieved the correct PatchHub game metadata and Title ID `434D0027`. The v0.14.0-test.21 Windows runner successfully rendered a synthetic JPEG through actual GDI+. Neither result explains why the owner's real app still shows a letter tile. This release is for **actual local evidence**, not a proven rendering fix.

## Verification and safety

Gated one-shot workflow must pass portable Go tests/race/vet, updater-v1 contract, icon generation, Windows PE build and checksums, real Windows UI smoke (now checking the cover controls, copy report and synthetic JPEG GDI+ rendering) and updater healthcheck **before** publishing. Same mandatory `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` assets, portable+installed updater contract, original README, privacy and offline support.

**Owner next step:** update to test.22, open American Idol (Prototype) in Patch Library, click **Check cover**, then **Copy details** and paste the text back. The initial status before pressing Check cover can also reveal whether the image is already cached but GDI+ cannot draw it.
