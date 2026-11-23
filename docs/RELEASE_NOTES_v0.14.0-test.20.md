# Magic Patch Desk v0.14.0-test.20 — missing game cover fallback

## Fix
- Recover game catalogue covers when Patch Hub's game card doesn't supply a usable cover URL but the image exists under `All-Patches/titles/<game folder>/<XBE Title ID>.jpg` or `.png`.
- Concrete reported example: **American Idol (Prototype)** → `434D0027.jpg`; the corresponding patch is `American Idol {720p} (PROTOTYPE) [SPPV].JMP`.
- Prefer existing local artwork and authoritative PatchHub image URLs. Use only the current game's known Title IDs, validate folder paths and reject ambiguous metadata instead of borrowing unrelated covers.
- Reuse trusted HTTPS, image verification and offline caching. Limit concurrent cover workers to three and cool down failed attempts to prevent repeated background load.
- Preserve original game files, all JMP patches, original PatchHub filenames, game source inspection, report privacy, official JMP icon, updater v1 and Windows 10/11 compatibility.

## Tests and known limitations
The gated release pipeline **passed** all Go tests, the race detector, vet, existing media-index fixtures, the updater contract, Windows PE cross-build and SHA256 verification, and native Windows GUI smoke and updater healthcheck. The GitHub prerelease assets were published only after the Windows checks succeeded. Run: https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45141316528.

Real-world PatchHub cover downloads and the user's interactive Windows catalogue are not guaranteed by synthetic tests; please check American Idol (Prototype) and other previously missing covers on your system. The app still shows a letter tile when no valid image is found. Current source patch identity / Title-ID checks remain strict.

## Test download
`MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` are the expected release assets. This is a **prerelease**, not a production sign-off.
