# Magic Patch Desk patch packs (`.mpdpack`)

A `.mpdpack` is an ordinary ZIP file containing `manifest.json` and one native patch per target file. Its goal is to preserve a single-file patch format's meaning while allowing a game-level set of changes to cover several different files. The archive **contains no original game files**, no built-in XBE assets, and no source text requiring interpretation.

Example:

```
GameFix.mpdpack
├── manifest.json
└── patches/
    ├── default.xbe.ips
    ├── driving.xbe.ips
    └── media/settings.cfg.ips
```

The manifest has `schema: "magicpatchdesk-pack-v1"`, title, optional author, origin JMP filename, optional JMP `game_version`, `title_id`, `expected_crc32` array, and an `entries` array. Each entry includes a unique `target`, a corresponding `patch` path, `format`, record count, `requires_source` flag, `patch_sha256`, and optional `source_sha256` and `result_sha256` fingerprints if the original file was present during export. Source/output checksums are validation only; they do not substitute for original bytes for a source-dependent patch.

Supported member patch formats: IPS (`.ips`), password-encrypted IPS (`.eips`), uncompressed/default-table VCDIFF (`.xdelta`) and BPS1 (`.bps`). All records belonging to one JMP target become one member patch. **Never merge two target filenames into one IPS/BPS/Xdelta stream.** The JMP itself remains a single multi-file `.jmp` when exported in JMP format.

### Original file requirements

- Offset-only JMP -> IPS/EIPS pack: **source optional**. The tool warns about lack of byte-level verification. Original JMP `version`-header CRC32 values remain in the pack manifest and are checked against `default.xbe` when the pack is applied.
- Search-and-replace or `APPEND` -> IPS/EIPS pack: **original for that file required** to resolve matches and append location.
- Any target -> BPS/Xdelta pack: **original for that target required**, as these formats are source-dependent.
- When an original is provided, the encoder re-applies the generated member patch and requires its output to be byte-identical to the original JMP result. Original and output SHA256 digests are stored in the manifest.

### Filename heuristics

The parser scans both individual `#` comment lines and the JMP `notes=` field for safe relative filenames with extensions, such as `settings.cfg`, `driving.xbe` and `Media/config.ini`. A per-record comment takes precedence. When only one filename is present in notes, records lacking explicit filenames inherit it. If notes contain several filenames, each active record must explicitly identify its target; ambiguous records fail closed rather than patching the wrong file. Obvious website URLs/domains are ignored. User approval is required for special-file application.

### Apply and security

The app validates unique relative member paths, ZIP member sizes, SHA256 checksums, optional original/result fingerprints, and the source-level BPS checksums. It asks the user to identify each source file and presents an all-or-nothing preview. It patches existing members of a supported XDVDFS image or writes external sidecar files to a new `_patched_files` folder. Originals are never overwritten. Multi-file ISO relocation remains limited to game-partition XISO; full-disc ISO growth past allocated sectors is refused. Individual patch files can be extracted manually from the ZIP if desired.

Current limitations: the Windows GUI has not been smoke-tested on a physical Windows desktop; `.mpdpack` is a Magic Patch Desk-specific packaging convention, not a widely standardized patch format; xdelta3 secondary compression/custom code tables are not supported. Batch tooling and arbitrary file creation inside an ISO are not provided.
