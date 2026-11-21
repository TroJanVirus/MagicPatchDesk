# Magic Patch Desk v0.13.0-test.1 — Game Outputs preview

**Preview / testing build — not a stable, fully verified release.**

This release adds a native **Game Outputs** workspace. Source inputs and output formats can be chosen independently:

- Accept a game-partition **XISO**, supported full-disc **ISO**, **Cerbios CCI**, an extracted game folder, or a standalone **XBE**.
- Select one or more game products: **original game XBE only**, **extracted game folder**, **rebuilt XISO**, **Cerbios CCI** (including split sets), and a **layout-preserving full-disc ISO** only for compatible full-disc ISO sources.
- Convert images without a patch or apply compatible **JMP**, **IPS/BPS/Xdelta/EIPS**, **.mpdpack**, and full-folder **Rocky5 PF** operations.
- Refuse incompatible and incomplete output combinations rather than silently omit named modified files. Preserve originals, stage new outputs without overwriting existing data, verify XDVDFS member hashes and compressed-image round-trips.
- Remember format choices by source category; drag and drop supported images and extracted folders.
- Show user-visible attribution for **Team Resurgent/Repackinator**, **Team Cerbios/libCCI/CCISharp**, **Rocky5/XBMC4Gamers**, and **JMP/PatchHub** references. Complete credit, scope, links and licences: [CREDITS.md](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/CREDITS.md). Repackinator GPL-3.0 source was **not** reused or bundled.

## Validation and known limitations

- Go test suite, race detector, static analysis and Windows x64 cross-build passed in [headless run #44913720896](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44913720896) on the implementation.
- This preview is additionally gated on a fresh Go test/race/vet run, updater-contract check, **native Windows-runner Go test execution**, and executing the **exact Windows release candidate** with `--mpd-healthcheck-v1` on Windows.
- **Not yet verified:** Interactive Windows 10/11 GUI behaviour (including selection and drag/drop), compatibility of rebuilt XISO/CCI images on real Xbox or Cerbios consoles, individual retail game files, managed self-update/rollback in an actual installed environment.
- XISO is a **game-partition image**, not a lossless reconstruction of a full-disc/Redump dump. Full-disc ISO output requires compatible full-disc ISO input and patches that fit existing sector allocations.
- No copyrighted game images, XBE files or third-party binary patching tools are included.
- Existing self-update protocol v1, canonical EXE name, SHA-256 release manifest, local patch data and privacy-related diagnostic handling are preserved.

## Installation and updates

Download **MagicPatchDesk-Windows-x64.exe** from this prerelease, and optionally check it against **SHA256SUMS.txt**. Existing v0.12.0-test.2 installations should detect this higher-versioned preview when preview updates are enabled; installing is optional, and the previous executable is retained for rollback. Use **Maintenance** to check and update.

**Please report Windows UI and game-image compatibility issues through the in-app Diagnostics workflow or [Issue #9](https://github.com/TroJanVirus/MagicPatchDesk/issues/9).** This is a test build for validation, not a claim that every supported game boots on original hardware.
