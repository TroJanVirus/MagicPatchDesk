# Magic Patch Desk v0.14.0-test.8 — Appearance, combined checks and PatchHub discovery

## Owner-facing changes

- **Appearance** is now a separate Settings panel. Four directly selectable themes (Midnight, Forest, Plum and Light) show multi-colour preview icons; changing a theme persists in the existing `settings.json`. Light mode uses neutral pale surfaces, ink-blue text and restrained teal accents. Legacy blue hardcoded panels, footer, tabs, controls and buttons are remapped for readability. Windows light title-bar mode follows the chosen theme.
- **Check + sync** combines release checking with GitHub JMP refresh and PatchHub website metadata refresh. The existing `Check at Start` toggle now controls the entire bundle. Turn it off for offline/manual operation; the cached local patch library still loads and the separate manual GitHub Sync button remains.
- **Find images** is now website-index-first. The old fallback incorrectly fabricated a folder from the GitHub JMP filename and stopped before image-index discovery when a title had multiple matching XBE revisions. Now the app first looks for images in the server's exact patch directory manifest, validates game/title ID/region/known revision/normalised patch name and only then falls back to a small bounded directory probe. No site-specific AMF exceptions. Ambiguous revisions are never treated as interchangeable.
- Failed screenshot lookups no longer silently return to `Find images`: the screen offers `Why no images?` with the underlying index/lookup failure. Only verified downloaded images appear.

## Image limitation requiring PatchHub hosting

The PatchHub game-list PHP page lists game covers, Title IDs and revisions, **not a complete list of per-patch JPG/PNG filenames**. A GitHub JMP filename is not an authoritative PatchHub website directory name. Even worse, an image folder can exist for a patch not present in GitHub: the supplied AMF Xtreme Bowling screenshot is NTSC Esppiral Widescreen, while upstream GitHub currently includes only Esppiral PAL Widescreen (plus a different NTSC 720p patch).

To reliably discover **all** arbitrarily named screenshots, the PatchHub website must publish `https://www.jayxbox.com/All-Patches/patchhub-media-v1.json`. Generate it from the real web-host `All-Patches/titles` directory with `scripts/build_patchhub_media_index.py`. Until it is published, the app cannot enumerate unlisted filenames; limited, verified conventional name probes remain supported. Never guess that another revision's images belong to a selected patch.

## Compatibility and release controls

Keep canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`, updater protocol v1, separate installed/portable update targets, existing user data and rollback executable. Existing sources are never modified in place. The release is a preview; test actual Windows theme legibility, 720p-height layout, startup toggle behaviour, and a live deployed media index. A passing Go/Windows cross-build is not a real hardware/interactive Windows claim.
