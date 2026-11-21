# Project state — native Windows first developer build

Last consolidated: 2026-11-21 (NZ). This document captures what is present in the source tree, not a promise of hardware compatibility.

## Implemented

- Go standard-library native Win32 Windows x64 UI with patch browser, conversion, settings, PatchHub catalogue integration and drag-and-drop local imports.
- GitHub JMP catalogue sync on launch, local cache, offline manual `Custom/` library.
- JMP parsing (metadata, CRC32/version, offset, first-match search/replace, append, default-disabled records) and group switches for identical case-insensitive comments.
- IPS parsing and generation; EIPS (custom password-protected IPS); uncompressed VCDIFF subset; stock-vs-patched XBE conversion.
- Original XBE Title ID/CRC32 inspection and patch compatibility checks.
- XDVDFS/XISO loading, default.xbe replacement, game-partition relocation for growth; full-disc growth refusal when beyond allocation.
- Multiworker bounded-memory ISO copying and patch indexing, output staging to preserve originals.

## Tested in this environment

- `go test ./...` passed, including `core` unit tests (2026-11-21).
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

## 2026-11-21 — v0.5 record filename routing and strict source prerequisites

Implemented in a test branch: filename.ext detection in JMP comment labels, safe per-file selection, multi-file native dialogs (locate / skip / cancel), explicit XISO member resolution and multiple XDVDFS member replacement, source preservation, and a disabled conversion button for input/output combinations requiring originals. Synthetic ISO multi-member and full-disc refusal tests pass. Manual Windows UI and commercial Xbox/XISO boot testing remain outstanding.

## 2026-11-21 v0.6: patch packs and UI hover

Multi-file JMP -> per-target IPS/EIPS/BPS/Xdelta packed into `.mpdpack` (ZIP manifest, per-target verification, no game contents). JMP remains one file. Filename detection now covers `notes=` plus comment, with safe ambiguity resolution. Pack loading/applying handles loose/ISO targets via explicit original file selection. Hover, cursor, pressed states and button animation are implemented in native GDI. Automated Go tests, race checks, vet and Windows x64 compilation pass. Real Windows GUI and Xbox hardware QA are pending.

## 2026-11-21 — v0.7 Rocky5 PF and all-variants pack

See docs/PF_FORMAT.md and docs/PATCH_PACK_V2.md. The PF parser/interpreter is distinct from JMP, and includes CP/HR/OF/SM/SR/MV/RN/RM. Its full-folder application stages a separate tree; ISO file operation changes are refused. New pack schema magicpatchdesk-pack-v2 includes original JMP plus all representable IPS/BPS/Xdelta/EIPS alternatives per named target and source/result checksums; v1 remains supported. Test go test ./..., race, vet and cross-build before releases.

## 2026-11-21 — v0.8 diagnostic reporting

Implemented local classified error and PF warning reports (no raw private paths, passwords, game bytes or patch payloads), Settings report controls and manual GitHub issue handoff. Go tests, race checks, vet and Windows x64 cross-build passed locally. Win32 clipboard/dialog and real-user issue submission still require manual Windows verification. No background telemetry.


## 2026-11-21 — v0.9 local diagnostic groups

Added deterministic group IDs by error type + operation + Title ID, backward-compatible aggregation of old reports, grouped Settings navigation, and user-initiated GitHub issue matching/search before issue creation. Local Go tests, race checks, vet and Windows cross-build pass; real Windows and GitHub UI verification still pending. No recurring or issue-triggered CI.


## 2026-11-21: v0.10 developer diagnostic triage

Native Diagnostics workspace indexes local failure groups and an opt-in
read-only GitHub issue snapshot; prioritizes public report frequency, recency
or open status; links a root issue and a fix PR by explicit body markers;
flags contradictory root-cause references instead of silently merging cases.
Local issue data is never automatically transmitted. Tests and Win32
cross-build passed locally; actual desktop/network testing remains pending.

## 2026-11-21 — v0.11 recent files and keyboard QoL

Added local-only MRU history for game source/patch/conversion inputs/output directories, no automatic submission of paths. New Recent tab with open, folder, copy, forget, clear and pagination. Cross-workspace shortcuts include Ctrl+O/P/F/R, Ctrl+1–6 and F5. All generated outputs are registered after successful writes, never when previewing or failing. Unit tests, race check, vet and Windows compilation passed locally; Windows GUI/Explorer workflows still need manual testing.


## 2026-11-21 — v0.12 maintenance foundation

Added self-install into `%LOCALAPPDATA%\Programs\MagicPatchDesk`, future-release check (stable/preview), canonical update asset + checksum/digest validation, separate helper with explicit v1 CLI protocol and previous-EXE rollback, no user data movement. Added opt-in redacted GitHub Issues upload via Windows Credential Manager and offline retry; auto-report OFF by default. Added pure Go CCIM LZ4 sector reader/writer and split support, CCI↔ISO maintenance tools and default-XBE CCI patch workflow. Windows GUI cross-build, tests, race and vet pass locally; real Windows update flow, GitHub authenticated posting and Xbox hardware CCI validation outstanding. Future release contract documented and checked in CI. No recurring GitHub Actions jobs.
