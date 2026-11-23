# JMP import deduplication and PatchHub screenshot galleries

**Implemented in v0.14.0-test.3 preview.** User-facing feature behaviour and limitations are documented here so future AI agents don't need earlier conversations.

## Import JMP: duplicate is a no-op

The existing **Patch Library → Import JMP** button and loose-file JMP/PF drag-and-drop now inspect the selected file and compute the full SHA-256 of all same-sized patches in both `%APPDATA%\MagicPatchDesk\Custom` and `%APPDATA%\MagicPatchDesk\GitHub` (including nested author folders). The comparison is of the **complete bytes**, not filename, game title, approximate hash or patch effect.

- If a byte-for-byte duplicate already exists, **do nothing**: no copied file, no overwrite, no patch selection reset, no unnecessary catalogue rescan. The bottom status/log line turns **red** and reads **"Duplicate patch found! No need to add."** A matching event is added to the local Game Manager → VERIFY LOG. No modal alert for this benign condition.
- If a differently named JMP contains the identical bytes, it is still considered a duplicate. A GitHub-synced duplicate is enough; the user does not need a redundant copy in `Custom`.
- If two files have the same filename but **different contents**, they are **not** duplicates. The new copy receives a safely numbered filename (e.g. `Patch (import 2).jmp`) instead of overwriting the original. Both remain accessible.
- The import rejects invalid or oversized patch files, symlinks and nonregular files; it parses the JMP/PF before any output write. Local files are created exclusively (`O_EXCL`), synced to disk and never replace existing patches.
- The existing `Import Folder` workflow still preserves imported folder hierarchy and artwork; it is distinct from the single-JMP duplicate behavior.

Source: `core/import_guard.go`, `core/catalog.go`, `cmd/magicdesk/main_windows.go`, `cmd/magicdesk/paint_windows.go`. Tests: `core/import_gallery_test.go`.

## PatchHub before/after screenshots, multiple images

The Patch Library now reads **patch-specific comparison screenshots**, not just the game cover. Select a game and its JMP; if screenshot metadata is found, the patch's details offer **View images (N)**. This opens an in-app, dark-background viewer showing two images side by side, explicitly labelled **Before** or **After**, with **Previous / Next / Close** to browse multiple screenshots. Screenshots are displayed with their original aspect ratio (no stretching/cropping); Esc closes the viewer.

### Sources and attribution

1. **Existing PatchHub metadata sync**: after validating the publicly returned PatchHub game-card HTML, the reader looks for image links or inline metadata under trusted `https://www.jayxbox.com/All-Patches/titles/...` paths. Only known-format JPEG/PNG/GIF images identifiable as before/after are accepted; the source path can identify a game revision/Title ID and patch folder. All images must resolve to the PatchHub HTTPS host, including redirects. No arbitrary third-party image requests.
2. **Imported PatchHub `titles/` folders**: for a JMP/PF contained within a patch-specific `titles/<game>/<version>/<patch>/` directory, look for independently named before/after screenshots under that patch directory (up to a bounded count). These are local and work entirely offline; imports preserve the artwork files.
3. Public image URLs are cached on demand under `%APPDATA%\MagicPatchDesk\Covers`, using the existing image size/dimension/content validation and bounded asynchronous downloader. Saved validated images can be viewed offline. Image downloads never include local game data.
4. Patch-to-image association is deliberately **conservative**: an exact patch-folder match or a unique identified patch match may attach screenshots. Ambiguous images or mismatched Title IDs are NOT silently attached to unrelated JMPs. Multiple before/after images remain grouped in a single scrollable viewer.

### Important availability limitation

The public **GitHub JMP repository** holds patch text files, not the PatchHub website's screenshots. A game cover is **not** a before/after screenshot, and the program must not claim to have such an image when none is discoverable. If the PatchHub page does **not expose** the photo URLs to its HTML/embedded metadata, this client cannot infer them or enumerate the server's private folders. User-supplied imported `titles/` folders remain the explicit offline fallback, and the UI should show no misleading screenshots instead of guessed images. Additional website metadata or a published gallery index would be needed for guaranteed coverage of every PatchHub screenshot. No images are copied into the project GitHub repository or compiled into the executable.

Source: `core/patchhub_gallery.go`, `core/patchhub.go`, `cmd/magicdesk/gallery_windows.go`, `cmd/magicdesk/art_windows.go`, `cmd/magicdesk/paint_windows.go`.

## Backwards compatibility and QA

- Existing `patchhub-index.json` cache files remain readable; the new `gallery_scan_version` triggers a one-time fresh PatchHub index request after upgrading. The ordinary 24-hour cache window then resumes.
- Existing `Custom/` and `GitHub/` patch libraries, `Covers/` gallery files and `%APPDATA%\MagicPatchDesk` data remain in place during upgrades. No new end-user dependency/runtime.
- Tests cover exact duplicates across both libraries, different filenames, no-overwrite for same-name new revisions, symlinks/invalid patches, safe PatchHub image URL extraction, multiple before/after pairs, image-to-patch ambiguity and offline local galleries.
- The Windows GUI smoke checks the import/gallery controls and opening/paging/closing the native viewer at a 1024x720 outer window without using copyright assets or network.
- **Manual QA still required:** confirm which comparison-image URLs the live PatchHub page currently publishes, download and render representative real images, and test click interaction on Windows 10/11. Automated parsing fixtures cannot certify live site coverage or retail Xbox patch compatibility.

## v0.14.0-test.25 selected-cover and gallery UX

The Patch Library header now includes a constant 101×100 px selected game cover surface; the image is aspect-fitted within it (not deformed or cropped), and larger via click in a bounded, aspect-fitted overlay. A magnifying-glass affordance appears when hovered. Normal list covers are also drawn fitted. The extra cover-specific debug buttons and copyable cover report were removed from the Patch Library by owner request; other privacy-preserving crash diagnostics continue separately in Settings. The comparison gallery retains two-up viewing, but Previous is disabled at first page and Next at last page, both disabled for one pair. Clicking the selected patch row again clears the JMP/PF selection without discarding source game or output-format choices, and renders the existing no-patch instruction.

Windows native smoke validates no stale cover debug controls, enlarged cover close, 101×100 preview hit, gallery boundaries at 3 and 2 photos, no-op disabled navigation, deselect, and scrollbars at 1024×720. Manual owner click testing remains recommended after update.

## v0.14.0 stable UI refinement (2026-11-24)

Patch names and region/author/TitleID metadata render within separate, nonoverlapping rectangles inside their 60px cards, aligned beside dynamic-width patch-type badges. The selected JMP/PF records and release records were not changed. Comparison image discovery remains automatic: the redundant `Find images` button no longer appears when a gallery is absent. The complete JMP notes remain available under `READ FULL NOTES`, now without the redundant "formatted + scrollable" suffix. Developer Diagnostics in the left sidebar is an opt-in Maintenance toggle, hidden on fresh installations; crash reports remain accessible from Settings independently. A bundled transparent user-supplied Patch Hub logo in the Home card links to the public website; it no longer implies patches missing from the app's synced catalogue. The titlebar is created invisible, themed first, and explicitly refreshed before normal window processing to address startup-frame artifacts.
