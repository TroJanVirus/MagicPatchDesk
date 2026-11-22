# Magic Patch Desk v0.14.0-test.9 — Patch folder images fixed

## Why test.8 missed real images

The owner supplied the actual PatchHub `patchhub-media-v1.json`, containing **477 valid JPG/PNG URLs across 253 distinct nested patch directories** (all URLs have the expected `All-Patches/titles/Game/Region_Version_TitleID/PatchName/Image.ext` structure). The prior application mistakenly required patch region/version/XBE Title ID to match gallery metadata even though images belong to the **patch folder** and may be named anything. These preconditions silently discarded otherwise valid images.

A second issue was a **24-hour HTML game catalogue cache**: `SyncPatchHub` would return early and skip newly published image-index content, so checking for updates could still show no images after uploading the manifest.

## Implementation

- Match images by real website game and **exact normalized patch directory label**, not by screenshot filename, region, Title ID or revision. The intermediate website version/release folders remain separate URL paths, but do not block displaying their images for the matching patch. JPG, JPEG, PNG (and existing GIF support) are accepted, with Before/After labels inferred optionally.
- Permit a shortened patch folder name (e.g., `720p Patch`) only when it uniquely identifies one JMP patch within the currently selected game. Distinct patch folders may never exchange images.
- Apply the same rule to both automatic PatchHub catalogue galleries and the asynchronous selected-patch image lookup; avoid useless conventional filename probes when a valid full index has no matching folder images.
- Refresh the lightweight media inventory independently of the 24-hour game-list HTML cache, reconciling newly added/removed photos and preserving offline cache when the website is unavailable.
- Add offline regressions for AMF Widescreen, Phantom Crash, arbitrary filenames, URL security, abbreviated folder titles, cross-region/revision attachment, wrong patch rejection and fresh image updates to cached game cards.
- No new image files or modifications to existing game/JMP files. The app downloads only trusted, validated image data on demand and caches it as before.

## Limitations / QA

The uploaded manifest structure has been inspected locally; live HTTP access to the hosted `https://www.jayxbox.com/All-Patches/patchhub-media-v1.json` could not be independently confirmed from this environment. If the host serves a 404, HTML instead of JSON, a blocked response, or the image URL fails to download/validate, the app will report that reason.

The public PatchHub site can host image-only patch directories without a matching GitHub JMP. Those do not automatically become editable/applicable JMP patch entries. For listed JMPs, patch folder names must match or have one unambiguous abbreviation. Region/revision-specific galleries of a same-named patch may contain multiple sets of images.

Preview v0.14.0-test.9 retains updater protocol v1 and canonical assets `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`. Automated tests and Windows cross-build are not a hands-on Windows GUI or real Xbox hardware test.
