## 2026-11-22 — v0.14.0-test.13: split-set XBE lookup

Selecting **any numbered CCI part** (`Game.1.cci`, `Game.2.cci`, etc.) now opens the complete contiguous part set, treats decoded 2048-byte sectors as **one read-only logical disc**, and parses XDVDFS metadata to locate the real `default.xbe`. Only the XDVDFS directory and requested XBE/file sectors are decoded for game identification or targeted patch verification, not the entire CCI. The title, Title ID and whole-XBE CRC32 are passed to the same Patch Library auto-matching logic used for a loose XBE or normal ISO. The source line displays the selected part out of the validated total.

The previous `OpenCCI`, `ReadSector` and full `ExtractCCI` methods remain supported. Full extraction/rebuilding still processes all sectors. Missing first/middle parts and bad CCI structure fail before presenting an identified game. As with all directory-only split set checks, a missing unknown trailing part cannot be proven absent if surviving data is internally consistent.

The app also recognizes **contiguous numbered raw ISO/XISO** parts via a separate sector-accurate virtual `io.ReaderAt`; see `docs/RELEASE_NOTES_v0.14.0-test.13.md`. It refuses to copy/compress a selected raw fragment as though it were the whole image.

# Cerbios `.cci` support (v0.12 preview)

Magic Patch Desk includes native (pure-Go, no external tool dependency) Cerbios `CCIM` v1/LZ4 sector decoding and encoding. The container is distinct from a conventional full-disc ISO. Reference: https://github.com/Team-Resurgent/libCCI/blob/main/FORMAT.md and https://github.com/purelogiccode/CCISharp/blob/master/docs/format.md.

A CCI part has a 32-byte LE header (magic `CCIM`, decoded size, index offset, 2048-byte block size, version 1, shift 2), a sequence of raw or LZ4-compressed 2048-byte sectors, then an index of `(sectors+1)` 32-bit entries. The high bit marks LZ4; the low bits store the sector offset / 4. Compressed sector data begins with a padding byte and has 0–3 trailing padding bytes. Each decoded sector must be **exactly 2048 bytes**.

## User workflows

- **Maintenance → CCI → ISO** extracts a `.cci` or the selected `.1.cci` split part to an uncompressed ISO. It does not alter the source.
- **Maintenance → ISO → CCI** compresses an uncompressed XDVDFS Xbox ISO into CCI; incompressible sectors are kept raw. Large images split into `.1.cci`, `.2.cci`, ... below FATX's per-file size limit.
- The **Patch Library** can select CCI input for **single-target default.xbe** JMP/IPS/BPS/Xdelta workflows. It decompresses to a temporary XISO, previews and applies through the existing XISO engine, encodes to a fresh CCI, decodes the result and checks byte-level integrity before marking success.
- **Multi-target JMP, PF file operations and .mpdpack** against CCI currently use the explicit CCI → ISO → patch → CCI workflow; direct CCI targets for these formats refuse to omit unsupported operations.

### Safety

- Headers, index offsets, sector sizes, LZ4 matches and split sets are validated; malformed sectors fail before use.
- Outputs are separate and an existing output is not overwritten. No source CCI, XBE, ISO, or split component is changed in place.
- CCI is a container, not a remedy for XISO full-disc sector-growth restrictions. The inner ISO writer retains existing XDVDFS allocation rules.
- These tests are software fixtures, not original Xbox/Cerbios boot validation; verify a patched CCI on actual hardware before publishing it as tested.
- Generic raw ISO/XISO input must contain a supported Xbox XDVDFS volume for conversion via the UI.
