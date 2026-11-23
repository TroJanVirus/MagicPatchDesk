# Resolving PatchHub game covers without a matching catalogue folder

Patch Hub's HTML game card title, JMP game-title header, and actual
/All-Patches/titles directory name can differ. An example confirmed by the
owner's test.22 diagnosis:

- GitHub JMP title: American Idol (Prototype)
- Patch Hub indexed display title: Miscellaneous-Prototype [American Idol]
- Verified shared Xbox Title ID: 434D0027
- Actual image: /All-Patches/titles/American Idol (Prototype)/434D0027.jpg

The **live** public Patch Hub page also displays the alias, so wiping the
user's local cache or re-syncing the same page is not a durable solution.

CoverCandidates has the following order:

1. Explicit exact cover URL from Patch Hub HTML, when available.
2. JMP/XBE game title directory, if no different Hub directory exists or if
   the different display name is independently corroborated by the same
   valid, nonzero eight-digit Title ID in a JMP and Hub version.
3. Alternative PatchHub display title directory from the same game record.
4. JPG and PNG extensions for each known (maximum 3) Title ID.

URL candidates are validated for safe folder names, correct path encoding,
and Title ID constraints; canonical titles are not guessed for unrelated
game records or mismatching IDs. The existing HTTPS downloader validates
image content, redirect host and size before showing it. A failed legacy
alias never prevents a valid canonical folder from being tried. These
are **cover lookups only**, not game-patch applicability decisions.

Current diagnostic UI in test.22 offers Check cover and Copy details.
Integration regression simulates an actual imported JMP and cached Hub alias
and requires the canonical JPEG to be the first candidate, even when the
Hub folder name disagrees. No website, JMP, media manifest or user cache
rewrite is needed for the matching fix.

The code is source candidate until the test build is verified and the user
checks artwork in the interactive Windows application.
