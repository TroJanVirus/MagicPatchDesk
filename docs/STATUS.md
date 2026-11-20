# QA status

2026-11-21 (NZ)

Confirmed:
- Go core unit tests (including race-enabled) PASS: JMP parsing, grouped labels and switches, CRC32, IPS/RLE, encrypted IPS, Xdelta/VCDIFF subset, XISO file relocation, full-disc growth refusal, PatchHub titles-folder metadata.
- Windows x64 cross-compile PASS: native Win32 PE32+ GUI binary.
- The Go and prior HTML engine's EIPS and VCDIFF uncompressed conversions passed cross-implementation round-trip checks during development.
- GitHub currently exposes 803 .JMP files in the public patch repository tree (snapshot at development time). The app obtains the real current count itself on each launch.

Unverified:
- Executable GUI on actual Windows 10/11 (no Windows runtime available in build environment).
- Real Xbox ISO and XBE patches across the complete GitHub corpus; there are synthetic fixture tests but not game-hardware tests.
- Windows SmartScreen reputation/code signing; current executable unsigned.
- Web-based PatchHub metadata scrape on a real Windows user's network (fails safely to cached index if unavailable).

Unsupported in first build:
- xdelta3 secondary-compressed/custom-table patch streams.
- Repacking a full-disc image after a patch expands an XBE beyond its allocated sectors.
- Compressed ISO formats, Xbox 360 XEX, automatic online patch attribution if an author omitted headers.
