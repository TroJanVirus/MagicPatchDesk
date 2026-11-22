# Magic Patch Desk v0.14.0-test.16 — ISO picker stability and filesystem hardening

The reported repeatable Windows crash was more specific than originally thought: **Patch Library → Phantom Crash → select its JMP → Open file → choose the game's ISO** caused the older app to exit. The owner reports that **v0.14.0-test.15 no longer crashes**, which is encouraging but not proof that every ISO failure path is safe.

## Additional issue found in the audit

The v0.14.0-test.15 fix moved XBE identification and patch matching onto a background-worker/UI-thread message boundary, but **`selectGameSource` still called `core.OpenXISO` synchronously on the Win32 UI thread** to guess whether an ISO was a full-disc image. This could block the native UI for complex disc files and bypass part of the previous stability work.

- **No filesystem scan during source selection.** The file dialog now returns immediately to the UI; XDVDFS directory discovery, default.xbe reading and output-kind classification all occur in the existing background worker. Only a validated result is published on the Win32 owner thread.
- **Output formats stay locked until verification succeeds.** Once the real disc layout is confirmed, game-partition XISO offers appropriate safe outputs, while a genuinely full-disc ISO may offer its preservation option. A failed or incomplete ISO never enables an output choice based solely on `.iso` or `.xiso` filename.
- **Stale scans remain rejected.** The previous generation/path guard continues to prevent results from superseded file choices enabling incorrect formats or changing the selected patch.
- **Game Manager manual-review sources** use the same background verification rather than bypassing this safety gate. Extracted folders and already validated loaded sources remain usable.

- **Extracted game folder safeguards:** exporting from an extracted folder is now held until its root default.xbe has passed identity validation. The Game Manager manual-review action supports both folders and disc images through the same asynchronous inspection route.

## XDVDFS parser hardening

- Replaced the per-directory recursive binary-tree traversal with an **iterative stack**, avoiding unbounded Go call-stack growth from deeply skewed or hostile directory tables.
- Bounded total directory metadata to **128 MiB** and all directory entries to **200,000** across the image, in addition to the existing per-directory-size and nesting limits.
- Added exact-length checks on `io.ReaderAt`; short reads are errors even if a reader unexpectedly returns a nil error.
- Require a regular disc-image file and recheck that a named member lies within the current file size before reading its bytes. This catches truncation or replacement between directory inspection and member access.

## Regression coverage

Added portable Go tests for:

- Identifying the correct `default.xbe`, Title ID and CRC32 from both game-partition XISO and full-disc ISO, with byte-for-byte verification that source images remain untouched.
- A 7,500-node skewed XDVDFS tree (traversal must complete without recursive stack growth).
- Truncated image members, corrupted root-directory lengths, malformed directory data and short `ReadAt` results.
- A 100-case collection of mutated volume and directory metadata that must not panic.
- Native Windows smoke assertions for **pending inspection → validated full-disc output vs game XISO output**, an invalid image refusing outputs, and a stale result never unlocking an unrelated image. Existing **selected patch → correct XBE** preservation smoke checks remain.

**Limitations:** No actual Phantom Crash ISO was provided for automated testing. Synthetic fixtures are not equivalent to running the Windows UI with that real image; this release does not claim to have reproduced the original crash or proved its root cause. The previously supplied recovery marker lacks a native fault trace. The owner must verify the exact ISO sequence on Windows, and a private WER dump/runtime log is still useful if any exit recurs.

The original game and JMP files are never overwritten, patch groups/toggles remain unchanged, and no patch is applied without explicit consent. The owner-written human README is **unchanged**. Updater protocol **v1**, exact canonical Windows asset `MagicPatchDesk-Windows-x64.exe`, and `SHA256SUMS.txt` remain compatible with all prior updater-capable releases.
