## 2026-11-22 — numbered image source handling (v0.14.0-test.13)

Source discovery now recognizes sequential `Game.1.cci` / `Game.2.cci`, `Game.1.iso` / `Game.2.iso`, and numbered `.xiso` variants. Selecting any part uses the entire validated, contiguous virtual disc for XBE identification and named-member reads. `core/cci_inspect.go` reads just required decoded CCI sectors; `core/split_image.go` maps raw split bytes into a logical random-access reader. `ExtractGameXISO` now also reads complete raw split sets, rather than directly opening only the selected part.

Raw split image outputs are deliberately limited to **rebuilt game-partition XISO / CCI, extracted folder and XBE**, as compatible with the source. Direct preservation as a full-disc/Redump ISO and one-fragment CCI compression are blocked. A full CCI game export continues to decode/extract the complete image to staging because it needs actual game files. See `docs/RELEASE_NOTES_v0.14.0-test.13.md` for exact filename convention and acceptance tests.

# Unified game-image outputs — historical design and current constraints


> **Status clarification (2026-11-22):** This file began as an implementation specification and preserves some *historical* acceptance checkboxes. Source-aware game outputs and the Windows Game Outputs workspace have since shipped in preview builds. The most recent published [v0.14.0-test.2](https://github.com/TroJanVirus/MagicPatchDesk/releases/tag/v0.14.0-test.2) passed Linux Go unit/race/vet, native Windows Go tests, actual EXE checksum/healthcheck, and Win32 UI paint smoke ([run #44923406249](https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/44923406249)). **Actual Windows user interaction, retail image round-trips and physical Xbox/Cerbios boot remain unverified.** For current state and next steps see [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) and [STATUS.md](STATUS.md); for reference licences see [CREDITS.md](../CREDITS.md). Do not treat an unchecked design item as proof the feature is absent from the current source.

## Current UI clarification — v0.14.0-test.5

The game-output engine in `core/game_outputs.go` is retained, but the **separate Game Outputs sidebar tab has been removed**. Single-game patching, source selection and multiple output-format selection now live together in **Patch Library**: choose an original game/extracted folder, choose a JMP or other supported patch, click **CHOOSE OUTPUTS** to select one or more formats and/or **Preserve current format**, then **APPLY PATCH**. Apply is genuinely disabled until a source, patch and explicitly chosen compatible output exist. Output is an independently verified package folder; original files are never modified. The old per-input-format remembered preferences are not silently applied to a newly chosen game, to avoid accidentally exporting or overwriting an unexpected format.

The output source pipeline can still export an independent extracted game directory, XBE, game XISO, Cerbios CCI, and full-disc ISO **only from an actual full-disc input**. The standalone `Game Manager` workspace is now displayed as **Batch Process Library**, retaining its separate bulk operations. See `docs/PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md` for the new native GUI, provenance and QA rules.

## User requirement

The selected **input** format must never lock the **output** format. A user who provides a game-partition XISO can choose an extracted folder, game-partition XISO, Cerbios CCI, or a patched game XBE alone. Full-disc/Redump-style ISO output is uncommon and should only be offered explicitly when a full-disc ISO was supplied and the image can be safely preserved.

This is separate from **patch-artifact formats** (JMP/IPS/BPS/Xdelta/EIPS/`.mpdpack`). A user may export patch formats, game formats, or both, without conflating a game image with a delta/patch archive.

Reference implementation for supported operations, CLI surface, safety considerations and compatibility: Team-Resurgent/Repackinator: https://github.com/Team-Resurgent/Repackinator. The external project's GPL-3.0 license must be reviewed before copying, linking, or redistributing any of its code or binaries. The current app is native Go with no bundled external runtime; prefer its existing Go XDVDFS/CCI functionality and use a carefully evaluated, optional Repackinator CLI adapter only if necessary. Never silently fetch or execute a third-party binary.

## Interaction model

1. Select source: `.xbe`, game-partition XISO (`.iso`), full-disc ISO, CCI (including split sets), or extracted game directory. Detect the actual disk layout using header/volume inspection, *not filename extension alone*. Show the source classification.
2. Select and preview JMP/IPS/BPS/Xdelta/PF/`.mpdpack` operations, including named/multi-file targets. Preserve all existing verification requirements. Independently choose **Game output** and optionally **Patch export**.
3. Game output choices as **multi-select**:
   - **Extracted game folder** (all original game files with selected changes applied).
   - **XISO** (new trimmed Xbox game-partition disc image, labeled XISO even if filename is `.iso`).
   - **CCI** (Cerbios compressed image; automatic FATX-safe multipart output).
   - **XBE only** (the *patched game executable*, not a generated ISO attacher/stub).
   - **Full-disc ISO** (explicit opt-in, only for supported full-disc ISO sources and byte-safe, layout-preserving operations; never imply arbitrary Redump-style ISO recreation from loose files).
4. Defaults: game-partition XISO source -> XISO output; CCI source -> CCI output; loose XBE -> XBE output; extracted game directory -> extracted folder. A full-disc ISO source may default to preserving ISO only when the patch can safely remain layout-preserving; otherwise offer XISO and explain why. Persist the last user choice **per source category** without silently choosing an unsupported combination.
5. Preview all artifacts, required space, loss of source-disc metadata, required temporary space, named-file scope, split outputs, and restrictions. If a source lacks the full game contents, disable folder/XISO/CCI with a reason; do not invent missing assets.
6. Apply selected patch operations **once** to a canonical staging representation, then create all chosen outputs from that same result. Use an original source checksum and per-target output hashes, verify produced outputs by reopening and comparing patched target bytes, and only publish successful artifacts. Original inputs and existing destination files are never overwritten.
7. Allow a preferred output directory and predictable names (e.g., `Game_patched/`, `Game_patched.xiso.iso` or `Game_patched.iso`, `Game_patched.cci`, `Game_patched.xbe`). Ensure a folder cannot recursively ingest itself while packing. Reserve temporary disk capacity for extraction/repacking and clean scratch content on failure.
8. Preserve all named changes. For an XBE-only output when a PF/JMP/pack also modifies other files, clearly flag that this would be **incomplete**; require explicit selection of a complete output or refuse, never present a successful full patch.
9. Patch exports remain independently selectable: JMP, IPS, BPS, VCDIFF, EIPS and the existing all-formats `.mpdpack`; preserve existing source-dependent encoding restrictions. Do not include copyrighted game binaries inside a patch pack.
10. Record non-sensitive diagnostic failures with stable categories; avoid logging actual game files, local paths, or passwords. No new background GitHub Actions.

## Pipeline and safety

`Source reader (XBE / XDVDFS / CCI / directory)` -> `verified virtual member tree` -> `patch plan` -> `staged patched member tree` -> independent `Folder / XISO writer / CCI writer / single XBE / safe full-disc ISO preservation`.

Current building blocks: `core/xiso.go` (XDVDFS index and file access), `core/xiso_multi.go` (named file patches), `core/cci.go` (native CCIM/LZ4), `cmd/magicdesk/cci_patch_windows.go`, and `cmd/magicdesk/{targeted,pack_apply,pf}_windows.go`. Current development changes: extraction and game-partition XISO construction now exist in `core/game_outputs.go`, and a multi-select output workspace exists in the native Windows UI. The automated Go tests and Windows cross-build pass, but real Windows GUI and console compatibility remain **unverified**. The earlier CCI direct-patch workflow remains intact; the new Game Outputs workspace uses extraction for complete multi-target outputs.

The XISO extraction/writer layer must reject traversal and symlinks/reparse-point escapes; enforce XDVDFS names, relative paths, sector alignment, size limits, ambiguous default-XBE selection and duplicate/case-insensitive names; use streaming I/O for large assets. Preserve nonpatched assets byte-for-byte. The writer must correctly construct XDVDFS directory trees, volume descriptors and offsets, including nested directories; validate the result using an independent reopened reader. Never fake full-disc sector structures; do not call a generated game XISO a pristine Redump ISO.

CCI output must verify a decode/hash roundtrip, preserve all modified files, and handle FATX limits/split naming. Compression should apply to the patched XISO, not to a whole full-disc ISO unless explicitly requested and verified. All output errors must be surfaced before declaring success.

## Acceptance tests

- [ ] Synthetic XISO -> extracted folder; byte-exact all members and correct tree.
- [ ] XISO -> patched XBE only; Title ID/CRC checks work; no attacher/stub substituted.
- [ ] XISO -> rebuilt XISO -> extracted folder byte-for-byte member comparison.
- [ ] XISO -> CCI -> extracted folder; verify all members, including named patched assets; test split-set boundary.
- [ ] CCI -> XISO, CCI -> folder, CCI -> patched XBE only; CCI originals remain unchanged.
- [ ] Folder -> XISO/CCI with nested directories, long names, Unicode/invalid names, duplicates and large assets.
- [ ] Full-disc ISO -> supported safe preserve-ISO and independently rebuilt game-partition XISO; unsupported growth fails with an actionable explanation.
- [ ] Multi-target JMP/PF/pack cannot silently export incomplete XBE-only results.
- [ ] Multi-select outputs from one patch execution; temporary failure/space exhaustion leaves originals and prior outputs untouched.
- [ ] Source format sniffing distinguishes full-disc ISO from game-partition XISO regardless of `.iso` extension.
- [ ] Preferences persist per input type, remain backwards compatible, and self-update protocol/assets are unchanged.
- [ ] `go test ./...`, `go test -race ./core`, `go vet ./...`, Windows x64 cross-build; Windows 10/11 GUI and Xbox/Cerbios boots explicitly marked unverified until tested.

## Delivery discipline

Implement in small, reviewed increments; keep `AGENTS.md`, `AI_HANDOFF.md`, `README.md`, `docs/PROJECT_STATE.md` updated as each output actually works. No generated game assets or binaries committed to source. Keep the mandatory v0.12+ self-update protocol and diagnostic/privacy contracts intact.
