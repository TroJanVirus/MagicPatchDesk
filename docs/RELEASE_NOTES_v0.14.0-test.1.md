# MagicPatchDesk v0.14.0-test.1 — Game Manager and field reports

**Developer preview:** no physical Original Xbox game boot or interactive Windows 10/11 session has been certified by this release process.

## New features

- **Game Manager:** read-only scan of extracted games, XISO and ISO files (CCI entries are review-first), independent game and patch matching using Xbox Title ID and original XBE CRC32. Uncertain games sort after exact matches. Set Auto, Review or Skip separately, choose an output format per game, or open that title in the existing Game Outputs workspace for manual approval.
- **Opt-in unattended batch:** only a single XBE-target JMP with matching Title ID and original CRC can be queued without manual review. The runner rechecks the original game and the JMP SHA256 immediately before applying; changed sources fail closed. Every output is a new verified package, existing output packages and originals are never overwritten.
- **Scrollable verification log:** local event viewer of preflight, processing, verification, errors and updates. No log content is sent to a server by merely viewing it.
- **JMP conflicts:** compare candidate JMPs' changed bytes against one original XBE in a hex representation. Identical writes and conflicting bytes are separated. Export a combined source-verified JMP with original author/source attribution; conflicting writes demand specific approval and the final output is independently re-applied and checked.
- **Image explorer:** inspect file and directory records from an XISO/ISO read-only and extract individual, independently rehashed files before doing any patching.
- **Patch field reports:** an exact patch hash + Title ID + original XBE CRC identifies the report subject. xemu and physical Xbox results are separate dated entries with working/partial/failed outcomes, no percentages. Users explicitly consent to post a field report as a public GitHub Issue; maintainers must review and commit reports to `confidence/reports.v1.json`. The app's usual update check also refreshes this curated public cache and announces new reports, even if the EXE is already current.

## Compatibility and safety

The standard permanent `--mpd-apply-update-v1` and `--mpd-healthcheck-v1` updater entry points, canonical `MagicPatchDesk-Windows-x64.exe` release asset and SHA256SUMS.txt remain unchanged. App data remains in `%APPDATA%\MagicPatchDesk`.

Existing save/patch copies and output packages are not touched. Game-source paths, game binaries, original patch payloads and local machine paths are not transmitted as confidence reports. Report history is user-submitted evidence, not a guarantee of hardware compatibility.

Documentation: [Library Manager and Confidence](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/docs/LIBRARY_MANAGER_AND_CONFIDENCE.md) and [Credits](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/CREDITS.md).

## Verification limits

Automated source tests, race detection, static analysis, Windows compilation, Windows-native tests, executable checksum and full native GUI paint smoke verification are required before this prerelease can publish. **Manual QA remains necessary** for actual Windows dialogs, batch workflows against commercial game assets, console/Cerbios booting, and large CCI images. Compatibility reports must not be treated as official product ratings.
