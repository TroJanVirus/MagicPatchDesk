# Magic Patch Desk v0.14.0-test.19 — smarter PatchHub comparison artwork matching

## Actual owner-reported mismatch

GitHub JMP: `Bad Boys - Miami Takedown {Widescreen} (NTSC) [Esppiral].JMP`

PatchHub image folder: `Bad Boys - Miami Takedown {Widescreen} (NTSC+PAL) [Esppiral]`

Both identify the same game, Widescreen patch, Esppiral author and Xbox Title ID `454D0016`. The GitHub JMP itself declares `region=NTSC`. The already-published `patchhub-media-v1.json` contains precise website image URLs but the app previously rejected them because it compared **entire normalized folder labels**.

## What changed

- Match explicitly multi-region website artwork folders to narrower, compatible JMP region labels (e.g. `NTSC+PAL` includes `NTSC` and `PAL`; `PAL+JPN` excludes `NTSC`).
- Use full normalized game/patch feature identity and author, and reject a conflicting known Title ID. An unrelated game, author, 720p patch or incompatible region cannot borrow the gallery.
- Prioritize **exact folder matches**, then validated regional aliases, then the existing unique-only abbreviated folder fallback. Duplicate candidates from the same region remain ambiguous, not arbitrarily assigned.
- Apply identical resolution to both normal Patch Library gallery cards and selected-patch **Find images** lookup. Keep the manifest-first path and all arbitrary JPG/PNG names and Before/After labels.
- No website update, GitHub JMP rename, altered hosted media JSON, image bundling or change to patch applicability/XBE validation is required.

## Verification

A new portable regression fixture embeds the exact supplied Bad Boys screenshot URL and tests parsing, cached gallery attachment, manifest-first selected-patch retrieval, ambiguous duplicate variants, explicit NTSC/PAL coverage, exact-match precedence and conflicting regions/authors/features/Title IDs. The release pipeline must pass core Go tests, the Go race detector, go vet, Windows x64 PE build, updater protocol-v1 checks, **actual Windows UI stress** and updater healthcheck before publishing.

## Compatibility and limitations

The original XBE/ISO/CCI/JMP files are never modified by media association. Existing cached site metadata and user settings survive. Updater protocol **v1**, the installed and portable update paths, `--mpd-apply-update-v1` / `--mpd-healthcheck-v1` and `MagicPatchDesk-Windows-x64.exe` plus `SHA256SUMS.txt` remain unchanged. This semantic mapping intentionally declines broad fuzzy guesses without corroborating identity. A live independent fetch of jayxbox.com from the development environment was unavailable; the real submitted URL, upstream GitHub JMP header and an actual index-format fixture establish the reported case, with final artwork loading to be checked on Windows.
