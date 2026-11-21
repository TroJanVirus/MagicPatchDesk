# Magic Patch Desk v0.12.0-test.2 — Self-updates, managed install, automated issue reports and Cerbios CCI

## New

- **Forever-update contract**: Maintenance page discovers the highest compatible official GitHub release (stable/preview channel), verifies GitHub asset digest and SHA256SUMS, launches a separate updater helper after exit, retains `MagicPatchDesk.previous.exe` and restarts without touching app data.
- **Managed app folder**: `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe` (user-triggered Install app). Existing loose EXE remains intact. Persistent settings, custom patches, GitHub cache, covers, reports and history remain under `%APPDATA%\MagicPatchDesk`.
- **Automated GitHub Issues**: opt-in user authorization, token stored in Windows Credential Manager, local-only redacted report queue and issue grouping; no background telemetry when off. Explicit Sync and Disconnect actions.
- **Cerbios CCI**: read/write CCIM v1 including LZ4, sector checks, split CCI; maintenance tools for CCI ↔ XISO; direct default-XBE patch pipeline to a new CCI with decoded-output verification.
- New documented mandatory release contract and build-time `verify_release_contract.sh` guard.

## Known restrictions

- Multi-target JMP, PF filesystem operations and patch packs inside CCI require **extract → patch XISO → recompress** for now. They are never silently skipped.
- Older v0.11 and earlier clients **cannot update themselves**; open v0.12 once and choose Install app. This is the unavoidable first migration.
- GitHub Issues auto-upload requires each user to connect a suitably scoped GitHub token, and is OFF by default. Reports become public when submitted; review permissions and content.
- EXE is unsigned; no release can guarantee a future version will run on every Windows system. The updater validates checksums and retains a previous EXE to help recover.
- Software tests and Windows cross-compilation do not establish real GUI, CCI/Cerbios console or user network verification.
