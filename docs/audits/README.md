# Canonical Patch Hub ↔ JMP audit snapshot — 23 November 2026

The Patch Hub source page listed **1,023 game titles** when this audit ran. The GitHub JMP repository contained **663 .JMP files**.

| Category | Count | What it means |
| --- | ---: | --- |
| Aligned | 473 | Exact matching site game-title and Xbox Title ID |
| Safe correction proposed | 85 | Matching Title ID and identical title words, differing only in punctuation or spacing |
| Manual review | 105 | Different wording or other uncertain association; cannot auto-correct safely |

Files in this directory:

- `title-audit.csv` — every JMP, original title, website title, Xbox Title ID, status, reason, original SHA256.
- `manual-review.csv` — uncertain mappings **and filename differences**. Filename differences alone do not mean a JMP header is wrong; prefixes such as `Halo 2 - DLC` can describe legitimate patch variants.
- `proposed-title-changes.patch` — **85 suggested `game-title=` header-only changes**. This is not applied to the magic-patches repository.
- `canonical-titles.v1.json` — full **proposed** game-title index from the live website (not an existing client cache schema, not deployed on the PHP server).
- `title-audit-summary.json` — machine-readable counts.

**Do not batch-apply all manual-review rows.** Example: `Author Unknown/AMF Xtreme Bowling {720p} (NTSC) [Unknown Author].JMP` declares ID `4B4E0002` while Patch Hub associates that ID with `Airforce Delta Storm`; that may be erroneous source metadata. This is deliberately not in the 85 proposed edits.

**Important:** The authoritative Patch Hub *display title* is not always the actual physical artwork directory. Do not rename the website's `All-Patches/titles` folders to match display strings. If the server webroot is available, use `--titles-root` to inventory exact cover paths into the JSON.

Source code and usage: [scripts/patchhub_title_reconcile.py](../../scripts/patchhub_title_reconcile.py), [docs/PATCHHUB_TITLE_ALIGNMENT.md](../PATCHHUB_TITLE_ALIGNMENT.md). A separate connected repository write grant or a local Git branch is needed to apply approved changes to `JayYardley/Xbox-Magic-Patches-by-Jay`.
