# Magic Patch Desk v0.6.0-test.1 — Multi-file patch packs & responsive controls

### New features

- JMP `notes=` field is now inspected as well as every active record comment for filenames with extensions (`driving.xbe`, `settings.cfg`, `Media/config.ini`, etc.). Record comments override global notes; ambiguous notes must be clarified on records. Website links are excluded; unsafe paths are rejected.
- A multi-file JMP can export a **single `.mpdpack` archive** containing one IPS/BPS/Xdelta/EIPS patch per target file and a JSON manifest. JMP itself is still a **single file**; no need to split it when staying in JMP format.
- Sources are **required only for source-dependent targets**. Offset-only JMP records can be converted into IPS/EIPS pack members without the original. Search/replace and APPEND need that member's original. BPS and Xdelta always need the original. Missing required originals disable the Convert action and show a Locate target originals button.
- Each generated member is independently verified against the original when provided. Manifest preserves JMP default-XBE CRC32 checks and stores optional SHA256 source, result and patch hashes. Game binaries are never packaged.
- Patch packs can be **loaded and applied** to loose files or supported XISO/full-disc ISOs. Source selection is explicit, all members are validated, and changes are written to separate output files. Original files remain untouched.
- Native Win32 buttons gain eased hover highlights, pressed-depth feedback, cursor changes and subtle shadows. Clicks now require a matching mouse-down and mouse-up target; clickable game rows, tabs and filter chips highlight on hover.
- Adds `docs/PATCH_PACK_FORMAT.md` explaining the archive schema, source rules and limitations.

### Validation

- `go test ./...` (file heuristics, source-required plans, offset-only IPS without original, BPS with originals, archive round-trips, checksum integrity, tampering rejection).
- `go test -race ./core`.
- `go vet ./...`.
- Go 1.23 cross-compile to portable Windows x64 Win32 GUI EXE; GitHub release build verification.

### Not yet verified

- Physical Windows 10/11 mouse interactions or dialogs.
- Booting updated Xbox game XBEs/XISOs on hardware or xemu.
- Third-party xdelta3 patches that use unavailable compressed/custom VCDIFF instructions.
