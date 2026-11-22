# v0.14.0-test.6 — JMP branding and PatchHub image discovery

## Exact owner-requested labels

- Sidebar: **Convert Patch** (formerly Convert & Create); **Batch Process** (formerly Batch Process Library).
- Native Windows caption: **Magic Patch Desk | Powered by Patch Hub** (formerly Xbox Patch Hub).
- The uploaded transparent **Jay's Magic Patch (JMP)** logo is embedded as a real Windows EXE icon resource. It appears in Explorer and in the window/taskbar; it is not a sidecar PNG.

The original uploaded `jmp.png` was used to prepare an antialiased transparent icon source. The text-encoded ICO asset is `assets/jmp-logo.ico.b64`, preserving portability through the GitHub UTF-8 source workflow. `scripts/build_windows_icon.py` (Python standard library only) decodes and SHA256-verifies it, creates an AMD64 COFF resource object `cmd/magicdesk/jmp_windows_amd64.syso`, and the Go linker embeds it. The generated `.syso` is ignored by Git and never required at runtime. To build from clean source:

```bash
python3 scripts/build_windows_icon.py
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w" -o MagicPatchDesk-Windows-x64.exe ./cmd/magicdesk
```

For Windows, use the equivalent `python` command and Windows shell variables. The native Windows executable smoke test now checks the actual window caption and that resource ID 1 is loadable via `LoadIconW`.

## Why screenshots were absent in v0.14.0-test.5

The PatchHub game-card HTML publishes titles, cover URLs, region and version entries, but **does not expose a full, patch-specific list of image filenames** in the metadata currently available to MagicPatchDesk. The gallery parser also only recognised names beginning with Before/After, so it rejected the **actual user-supplied file**:

`https://www.jayxbox.com/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Phantom%20Crash%20%7BWidescreen%7D%20%28NTSC%29%20%5BJay%5D/1.%20Before.jpg`

The new parser correctly classifies `1. Before.jpg` as Before #1 and `2. After.png` as After #2. It also recognises `Before1`, `after-2`, upper/lower-case JPG/JPEG/PNG/GIF extensions and arbitrary patch-specific image filenames. Unknown names are labelled **Screenshot** instead of falsely calling them a Before/After. Images are matched to the specific game, patch directory, Title ID, region and published game revision; no arbitrary game-cover substitution.

### What the app can discover without a website change

1. **The exact Phantom Crash image supplied by the owner** is a trusted, public metadata seed attached to its correct USA/version-5 JMP if the game/version exists in the current PatchHub index. This source URL is *known*, but a completed live download is not asserted until the client successfully receives and verifies its image bytes.
2. Selecting a patch initiates a **bounded, asynchronous lookup for that one patch**. The app derives a unique website directory using the game name, JPG/JMP filename, region, 8-digit version and Title ID. If several matching revisions are plausible, it declines to guess. It first looks for any directory-provided JPG/PNG links, including arbitrarily named files; if listing is unavailable, it tests a fixed small list of common numbered Before/After JPG/PNG names with HEAD. At most 20 guessed URLs for the one selected patch, not a recursive crawl across the entire server. Attempts are session-cached for 12 hours per patch, with an explicit Find images action for retry.
3. Found images are validated by the existing HTTPS-only allowlist, bounded 5-MiB download, actual JPEG/PNG/GIF image decoding and dimension checks, then cached under `%APPDATA%\MagicPatchDesk\Covers` and shown in **View images (N)** alongside multiple comparison photos. Failed images are not passed to the GDI+ renderer.
4. The previous gallery still supports locally imported PatchHub `titles/<game>/<region_version_TitleID>/<patch>/` photo folders, including files with arbitrary PNG/JPG names.

### What cannot be inferred from one URL alone

An arbitrary filename such as `Extra comparison C.png` cannot be predicted from `1. Before.jpg`. On a site with directory listing disabled, there is **no safe way to enumerate every other JPG/PNG in a directory from the URL scheme alone**. Previous statements implying full website screenshot coverage were premature.

To solve this for **all games and all existing images**, the app now supports an optional server-generated **public media index** at:

`https://www.jayxbox.com/All-Patches/patchhub-media-v1.json`

Schema:

```json
{"schema":"patchhub-media-v1","images":["https://www.jayxbox.com/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Phantom%20Crash%20%7BWidescreen%7D%20%28NTSC%29%20%5BJay%5D/1.%20Before.jpg"]}
```

The included `scripts/build_patchhub_media_index.py` generates that JSON from the *actual public website All-Patches/titles folder tree*, discovering arbitrary filenames without making any predictions. It only includes proper JPG/PNG/GIF magic signatures of files in the expected versioned patch hierarchy; it skips executables, symlinks, internal paths and files over 5 MiB. Example command for the **web host or an accurate local copy of its public tree**:

```bash
python3 scripts/build_patchhub_media_index.py /path/to/webroot/All-Patches/titles
```

The output defaults to `/path/to/webroot/All-Patches/patchhub-media-v1.json`; publish it on PatchHub and regenerate it as website media changes. With this index present, the app downloads it during the ordinary PatchHub metadata refresh (once per 24 hours), validates every path/host and attaches all media to the correct JMP. With the index absent/404, the app still works offline and uses the bounded per-patch discovery instead. Existing metadata caches remain readable; the version bump triggers one automatic gallery-aware refresh.

**Important website access boundary:** This project has GitHub repository access but does **not** have authenticated shell/FTP/deployment access to `jayxbox.com`. The source for the media-index generator is committed and tested, but **the JSON cannot be published to that website by this agent**. Until the media index is deployed or the site exposes directory listings, complete arbitrary-name discovery across all games cannot be claimed.

## Crash reporting note

The user checked Reliability Monitor (`perfmon /rel`) for the two v0.14.0-test.5 CTDs and found no Windows entry. This does **not** invalidate the private app runtime trace of the first native `0xC0000005` exception during `DispatchMessageW`; Windows does not guarantee an Application Error report for every process termination, particularly one managed by a runtime. The program's own private crash traces remain the authoritative evidence. This version records the `ui_callback` marker after entering the normal event-loop stage, so an unclean exit in dispatch/painting no longer misleadingly says only `startup`. **The precise root cause of the two crashes is not established and not claimed fixed.** Preserve Issue #19: https://github.com/TroJanVirus/MagicPatchDesk/issues/19. No user data, patches, original game assets or updater protocol were changed.

## Verification / handoff

Tests: `core/patchhub_media_discovery_test.go`, existing `core/import_gallery_test.go`, `scripts/test_patchhub_media_index.py`, native Windows `--mpd-ui-smoke-v1` brand/icon checks. Python resource generator uses no external SDK and was independently validated to yield actual Win64 `RT_ICON` and `RT_GROUP_ICON` resources in a cross-built PE; final shipping binary still requires full gated CI and user's real Windows startup test.

Never claim all website image URLs were discovered or that the real CTD was fixed by synthetic tests. Retain the unchanged updater v1, user data layout, original-XBE/ISO no-overwrite policy, one-shot GitHub Actions release gates and repo-first fresh-conversation handoff.

## 2026-11-23 — Region-superset gallery aliases (test.19)

The PatchHub manifest inventories real screenshot URLs and is not the bug. Its website patch-directory label may legitimately differ from the GitHub JMP filename for the **same** game, modification and author: `Bad Boys - Miami Takedown {Widescreen} (NTSC+PAL) [Esppiral]` vs GitHub `Bad Boys - Miami Takedown {Widescreen} (NTSC) [Esppiral].JMP`. If no exact folder owns the gallery, a multi-region website label can cover a narrower explicitly listed region variant only when complete normalized game/feature and author agree and known Title IDs do not conflict. Exact matches take priority, unique abbreviated names retain existing handling and same-region duplicate candidates fail closed. This affects **gallery association only** and never patch applicability. It uses existing `patchhub-media-v1.json` with no website edits; new tests validate both cached and manifest-first retrieval.
