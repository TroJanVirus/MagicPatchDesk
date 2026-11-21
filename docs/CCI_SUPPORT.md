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
