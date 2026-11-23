# Missing PatchHub game cover fallback

The exact game-card image URL is first priority. Some game art is hosted under `/All-Patches/titles/<game-folder>/<XBE Title ID>.jpg` without a usable image URL in the cached HTML index. Example: American Idol (Prototype), Title ID `434D0027`.

When game artwork is missing, the catalogue uses the existing PatchHub game-folder title (or the catalogue title when there is no matched Hub game). It constructs only a bounded `.jpg` and `.png` set from unique, valid eight-hex-digit Title IDs already in that game's patches/Hub versions, refusing unsafe folder names and more than three conflicting IDs. The canonical Hub image URL still wins. No cross-game fuzzy-art matching is permitted.

Game covers are downloaded only on demand by the existing trusted HTTPS PatchHub downloader, whose image parser checks dimensions and bytes, bounds the size, validates redirects and caches valid files for offline use. At most three worker goroutines can exist, regardless of catalogue scrolling; failed groups cool down for 30 minutes. Existing local imported cover files override these requests. Patch data, source game files and PatchHub website filenames remain unchanged.

Verification: isolated Go helper tests passed, a catalogue-cache integration regression is in the branch. Full Go/Windows CI, actual interactive UI and live website download remain unverified. No release or GitHub Actions auto-run was created.
