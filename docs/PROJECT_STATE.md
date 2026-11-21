# Project state — v0.2.0 test branch (post-v0.1.0 baseline)

Last consolidated: 2026-11-21 (NZ). This document captures what is present in the source tree, not a promise of hardware compatibility.

## Implemented

- Go standard-library native Win32 Windows x64 UI with patch browser, conversion, settings, PatchHub catalogue integration and drag-and-drop local imports.
- GitHub JMP catalogue sync on launch, local cache, offline manual `Custom/` library.
- JMP parsing (metadata, CRC32/version, offset, first-match search/replace, append, default-disabled records) and group switches for identical case-insensitive comments.
- IPS parsing and generation; EIPS (custom password-protected IPS); uncompressed VCDIFF subset; stock-vs-patched XBE conversion.
- Original XBE Title ID/CRC32 inspection and patch compatibility checks.
- XDVDFS/XISO loading, default.xbe replacement, game-partition relocation for growth; full-disc growth refusal when beyond allocation.
- Multiworker bounded-memory ISO copying and patch indexing, output staging to preserve originals.
- **New in v0.2.0 test:** dry-run XBE/ISO preview before writes, with predicted CRC32 and per-record byte offsets, and confirmation before applying.
- **New in v0.2.0 test:** expand repeated JMP comment groups and toggle individual records; mixed groups clearly indicated.
- **New in v0.2.0 test:** worker count and catalogue filter persisted to a local JSON settings file. Verification override deliberately resets OFF on launch.

## Tested in this environment

- `go test ./...` and `go test -race ./core` passed, including new patch preview, mixed-group, and JSON preferences unit tests (2026-11-21).
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -s -w' ...` succeeded.
- A fresh cross-compiled executable matched the prior distributed binary byte-for-byte by SHA-256.

## Not verified

- The GUI has not been executed on real Windows 10 or 11 here.
- Patched commercial games have not been hardware-tested on the original Xbox or xemu in this session.
- Full real-world compatibility across the live JMP repository has not been exhaustively established.
- This EXE is not digitally signed and may trigger Windows reputation warnings.

## Release/repository policy

- Source, tests and documentation belong in Git; binaries belong in Releases.
- Avoid GitHub Actions workflows unless explicitly requested, to minimise CI minutes.
- Preserve patch authorship, backups and user data.
- For contributor context, read `AGENTS.md`, `AI_HANDOFF.md`, and `README.md`.

## Branch/release layout

- Baseline source: `main` at `2ca03edcc227174b4e252ebe5657142f5666ae29`.
- Initial portable release: `release/v0.1.0` runs a **single** GitHub Actions job that creates the GitHub Release `v0.1.0` with built EXE and SHA256SUMS. No recurring jobs.
- Current test features: `feature/dry-run-record-controls-settings` proposed as v0.2.0 test, not merged into baseline until Windows GUI QA.

## 2026-11-21: BPS / IPS encoding and patch-to-patch conversion

Version v0.3.0-test.1 adds BPS1 codec and source/target/patch CRC checks, optional standard IPS RLE encoding, a direct source-patch picker in Convert & Create, all five output formats, and 25 synthetic conversion combinations across five input formats. Original XBE remains mandatory. Xdelta3 compatibility is still limited to uncompressed VCDIFF streams; do not claim universal xdelta3 support.
