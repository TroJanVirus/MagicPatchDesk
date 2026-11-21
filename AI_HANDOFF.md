# MagicPatchDesk — development handoff

## Goal

Create a fast, visually approachable *native* Windows application that browses the Xbox patch catalogue, applies JMP/IPS/EIPS/limited VCDIFF patches to XBEs and supported XISOs, and creates/converts patches. The original HTML proof of concept is **not** the target UI.

## Authoritative sources

- Application and patch engine: this repository, `core/` and `cmd/magicdesk/`.
- Public JMP corpus: https://github.com/JayYardley/Xbox-Magic-Patches-by-Jay
- Website catalogue metadata and artwork: https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php
- This project's current capabilities and limitations: `README.md` and `docs/PROJECT_STATE.md`.

## Architecture

| Area | Files | Notes |
|---|---|---|
| Windows GUI | `cmd/magicdesk/*_windows.go` | Native Win32/GDI+ UI, input/output dialogs, navigation, artwork |
| Patch formats | `core/jmp.go`, `ips.go`, `eips.go`, `xdelta.go`, `formats.go`, `convert.go` | No external runtime; VCDIFF subset only |
| XBE metadata | `core/xbe.go` | Title ID and original XBE CRC32 |
| ISO/XISO | `core/xiso.go` | XDVDFS image handling; defensive allocation limits |
| Catalog and sync | `core/catalog.go`, `core/patchhub.go` | GitHub patch sync, local caches, optional PatchHub metadata |
| Tests | `core/core_test.go`, `testdata/interop/` | Synthetic fixtures and JS/Go-format interoperability |

## Invariants

1. Never modify the user's input image or executable in place.
2. JMP record groups share a checkbox by case-insensitive comment, but individual records remain ordered; `[-]` starts disabled.
3. Verify a JMP's specified CRC32 values against the *original XBE bytes* (even when found inside an ISO).
4. Preserve local `Custom/` files during synchronization of the GitHub catalogue.
5. Keep all patch operations usable without internet; online sync and cover art are optional.
6. Make unsupported formats and unsafe ISO growth explicit errors, not silent success.
7. Keep Windows distribution portable; unsigned developer executables may trigger SmartScreen, and code signing is not solved by bundling a file.

## Known gaps

- Windows UI and console-boot compatibility have not been independently validated on real hardware.
- Xdelta3 secondary compression and custom VCDIFF instruction tables are unsupported.
- Full-disc ISO expansion is restricted; arbitrary files, compressed images, and ISO repacking are unsupported.
- No GPU patch acceleration; worker count applies to indexing and ISO copying, not sequential JMP records.
- No automated game-image regression corpus (copyright concerns); add synthetic XDVDFS and JMP tests as improvements are made.
- No executable auto-updater, presets/undo journal, batch conversions or complete game-asset editor yet.

## Recommended next actions

1. Verify launch, rendering, file dialogs, DPI and drag/drop on a clean Windows 10 and Windows 11 install.
2. Add end-to-end fixture tests from real JMP metadata (without redistribution of game bytes); test CRC32 matching and group controls.
3. Improve catalogue UX and safe/explicit ISO compatibility detection.
4. Expand image formats only with isolated parsers and round-trip tests.
5. Release signed Windows binaries via GitHub Releases when ready, rather than committing binaries to source control.

Never claim a Windows GUI run or Xbox boot passed based only on Linux unit tests or Windows cross-compilation.

## v0.5 developer handoff — 2026-11-21

New `core/targets.go` parses filename-with-extension record comments into target names, enforces path safety and offers `Patch.ApplyToTarget`/`PreviewTarget` for named files. Single-file `Apply`/`Preview` must reject enabled named targets via `CheckSingleFileOnly`.

`cmd/magicdesk/targeted_windows.go` intercepts multi-target JMP operations before the regular XBE preview/apply flow, prompts Yes=locate/use ISO member, No=skip target, Cancel=abort, then builds separate output copies after per-file preflight. `core/xiso_multi.go` updates multiple existing XDVDFS directory entries in an output image, allowing grow/relocate only in a game-partition XISO. The UI blocks source-required conversion, warns about conversion limitations, and does not allow a multi-target JMP to be collapsed into one IPS. Test synthetic fixtures first; still needs physical Windows testing, xemu and hardware game validation. Avoid distributing copyrighted game files.

## 2026-11-21 v0.6: multi-target patch packs / filename notes / reactive Win32 UI

- `core/targets.go`: `FileTargetsInText`, `Patch.RecordTarget`. Every comment and `notes=` is scanned for filename.extension references. Ambiguous notes require explicit per-record target. URL links ignored; unsafe paths fail.
- `core/pack.go`: `PlanJMPPack`, `BuildJMPPatchPack`, `ReadPatchPack`, `ApplyPatchPack`. `.mpdpack` is ZIP + manifest + per-target IPS/BPS/Xdelta/EIPS; optional source SHA256/result hash; default-XBE JMP CRC32 retained in manifest. No original game data in archive. Full specification: `docs/PATCH_PACK_FORMAT.md`.
- `cmd/magicdesk/pack_windows.go`: source-aware per-target pack conversion, per-target file selection; disabled Convert until required originals available.
- `cmd/magicdesk/pack_apply_windows.go`: explicit full-pack application via existing XDVDFS patching / separate external copy pipeline; refuses missing targets.
- `cmd/magicdesk/main_windows.go` and `paint_windows.go`: custom eased mouse hover, clicked-depth states, real hand cursor and responsive game/chip/sidebar highlights. Do not conflate Win32 GDI UI with CSS; no browser used.
- Build and run core tests on Linux; manual Windows 10/11 and xemu/hardware validation outstanding.

## 2026-11-21 — v0.7 Rocky5 PF and all-variants pack

See docs/PF_FORMAT.md and docs/PATCH_PACK_V2.md. The PF parser/interpreter is distinct from JMP, and includes CP/HR/OF/SM/SR/MV/RN/RM. Its full-folder application stages a separate tree; ISO file operation changes are refused. New pack schema magicpatchdesk-pack-v2 includes original JMP plus all representable IPS/BPS/Xdelta/EIPS alternatives per named target and source/result checksums; v1 remains supported. Test go test ./..., race, vet and cross-build before releases.
