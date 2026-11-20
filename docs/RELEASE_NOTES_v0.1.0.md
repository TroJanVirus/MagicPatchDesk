# Magic Patch Desk v0.1.0 — Native Windows Developer Preview

First native Windows release of Magic Patch Desk, built from the original source import.

## Included

- Portable Windows x64 Win32 application (no .NET, Python, Electron or browser required).
- Visual game/patch catalogue sourced from public GitHub JMP collection and optional PatchHub metadata.
- Startup GitHub sync with cached offline fallback and local JMP imports.
- JMP support: ordered search/replace, offset writes, APPEND, CRC32/Title ID verification, `[-]` disabled records, grouped identical comments.
- XBE patching and original Xbox XDVDFS game-partition XISO / supported full-disc ISO patching.
- Conversion of stock/patched XBEs to JMP, IPS, custom encrypted EIPS, and limited uncompressed VCDIFF/Xdelta.
- 1–8 workers for ISO copying and catalogue indexing; originals remain unchanged.

## Compatibility limits

- Not a general ISO rebuilder; compressed CCI/CHD unsupported.
- Full-disc ISO growth past allocated XBE sectors is refused.
- VCDIFF support is limited; many external xdelta3 files require features not implemented here.
- Windows GUI and Xbox-console behaviour have **not** been manually tested.
- The unsigned executable may produce SmartScreen warnings.

## Verification

The source release is compiled using Go 1.23 and a single scoped GitHub Actions Windows cross-build on Ubuntu. The workflow runs `go test ./...` before building and includes the EXE SHA-256 checksum as a separate asset.

## Next

A separate feature branch contains a preview-before-apply flow, individual JMP record controls and persistent safe UI preferences. Those features are **not** part of v0.1.0 until reviewed/tested.
