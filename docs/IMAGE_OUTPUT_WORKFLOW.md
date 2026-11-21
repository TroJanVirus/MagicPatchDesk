# Unified game-image outputs (source implemented; QA pending)


> **Development status — 2026-11-22:** Native source for extraction, XISO writing, CCI export, source-aware multi-output staging, and patch-aware Windows controls has been committed. Synthetic tests have been authored but not executed; no cross-build, native Windows 10/11 GUI smoke test, xemu test, or physical Xbox/Cerbios boot has been performed. This is not a verified release. Full credits and licences: [CREDITS.md](../CREDITS.md).

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

Current building blocks: `core/xiso.go` (XDVDFS index and file access), `core/xiso_multi.go` (named file patches), `core/cci.go` (native CCIM/LZ4), `cmd/magicdesk/cci_patch_windows.go`, and `cmd/magicdesk/{targeted,pack_apply,pf}_windows.go`. Current development changes: extraction and game-partition XISO construction now exist in `core/game_outputs.go`, and a multi-select output workspace exists in the native Windows UI. They remain **unverified** until Go tests and console/Windows validation run. The earlier CCI direct-patch workflow remains intact; the new Game Outputs workspace uses extraction for complete multi-target outputs.

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
