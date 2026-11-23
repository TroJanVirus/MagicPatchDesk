# Magic Patch Desk v0.14.0-test.23 — verified Title ID cover alias fix

## Root cause confirmed from user's test.22 diagnostic

The user provided the complete in-app report showing the app tried only `Miscellaneous-Prototype [American Idol]/434D0027.jpg` and `.png` and received HTTP 404. The image is really under `All-Patches/titles/American Idol (Prototype)/434D0027.jpg`. Both names correspond to the same XBE Title ID `434D0027`.

**The live Patch Hub HTML itself displays `Miscellaneous-Prototype [American Idol]`,** so clearing/re-syncing the locally generated `patchhub-index.json` would not correct the incorrect fallback by itself. The site's optional `patchhub-media-v1.json` is independent, primarily for patch-specific screenshot paths, not the source of this game-cover mismatch.

## Actual code change

- Keep explicit validated website image URLs first.
- When GitHub's JMP title differs from the Patch Hub display title, permit its canonical game folder name only after the JMP and Hub independently match on the same valid eight-digit Title ID.
- Try the canonical JMP folder's `.jpg` and `.png` covers before probing the alternate Hub display-title folder.
- Preserve existing bounds (maximum three unique Title IDs, two directories, two extensions), safe path validation, HTTPS image decoding, offline cache and per-game `Check cover` / `Copy details`.
- No website, JMP or cached index file needs manual renaming. Patch applicability, CRC/region safety, original games, and updater protocol v1 are unchanged.
- New Go regression reproduces **exactly the owner's test.22 diagnostic**, verifies correctly ordered URLs, rejects unrelated aliases without corroborating Title ID, and tests a real imported JMP + mismatched cached Hub index.

## Gated testing

One GitHub Actions workflow runs the full Go suite, race detector and vet, updater contract and official icon, native Windows x64 GUI cross-build, original Xbox source-switching/UI smoke and updater healthcheck, then publishes the canonical EXE plus SHA256SUMS.txt **only on success**. It also probes the actual public American Idol JPEG in that same run, verifying that the new first URL is downloadable as an image.

**Owner field verification remains required:** install test.23 and revisit American Idol (Prototype). The correct live URL and regression evidence strongly support this fix, but automated CI is not a substitute for the user's interactive Windows confirmation.

