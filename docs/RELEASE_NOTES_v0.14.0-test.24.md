# Magic Patch Desk v0.14.0-test.24 — safe BeatDown catalogue grouping

## What the owner's evidence showed

Three actual upstream JMPs refer to the same Xbox Title ID `43430015`, revision `CC-021`, region `NTSC`:
- Nemesis2000 Widescreen: `game-title=BeatDown - Fists of Vengeance`
- FrankyBuster (icup321) Widescreen: `game-title=BeatDown - Fists of Vengeance`
- Silverrock 720p: `game-title=BeatDown: Fists of Vengeance`

The live PatchHub game index lists **BeatDown - Fists of Vengeance**. The previous catalogue grouped JMPs by exact lowercase `game-title` and displayed two game entries. Silverrock's colon variant never received the canonical hyphen cover.

## Fix

- Group punctuation/whitespace-only name variants **only when every patch in each group has the same valid nonzero Xbox Title ID**. Use the actual sequence of title words, not character-stripped fuzzy matching.
- Preserve all three separate patches, their authors, filenames, records, CRC/revision and region safety. No source JMPs, hosted assets, XBE/ISO/CCI games or website filenames change.
- Choose deterministic display title: local cover first, otherwise greatest number of available patches, otherwise lexical tie-break. BeatDown displays **BeatDown - Fists of Vengeance** with all three patches.
- Preserve verified alternative JMP game titles as additional bounded cover URL candidates in case the site's actual artwork directory uses an alternate spelling.
- Prevent unrelated games sharing a reused Title ID from being arbitrarily assigned the wrong PatchHub metadata via nondeterministic Go map iteration: ID-based attachment is permitted only for a **unique catalogue game**.
- Conservative tests: real three-header BeatDown fixture; different-ID lookalikes; unrelated same-ID titles; token-boundary collisions; missing IDs; mixed IDs; ambiguous ID-to-Hub association.

## Verification and compatibility

The single branch-scoped GitHub Actions run tests Go suite/race/vet, website media manifest fixture, updater protocol v1, official JMP icon, Windows 64-bit build and checksum, and native Windows `--mpd-ui-smoke-v1` plus updater healthcheck. A test release is published **only after those gates pass**.

**Field test:** Search BeatDown after updating to test.24. Expect one game entry with all three patches and its previously working cover, while unrelated games remain separate. Confirm with the owner before marking the reported issue resolved.

Expected canonical assets: `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`.

