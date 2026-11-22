# Magic Patch Desk v0.14.0-test.4 — Crash recovery and Patch Library usability

**Developer preview.** Unsigned Windows x64 application. This release does not certify bootability on retail Original Xbox consoles or guarantee recovery from every possible native exception.

## Crash diagnostics — no more silent abnormal exits

- The previous reporter only created JSON for *handled* errors. It never intercepted fatal Go/Windows crashes or forced termination; that is why the Phantom Crash test could terminate without an available report.
- Durable session markers and a redirected private runtime stderr log detect unexpected process termination. On the next launch, an **unexpected_exit** category becomes a standard privacy-filtered diagnostic and a private human-readable evidence log, even if no stack trace survived.
- Main GUI / callback / selected worker panics also produce private stack evidence and a structured **unhandled_panic** diagnostic before the app exits.
- A persistent red footer notifies the user about unseen recovered crashes and links to **Settings → Problem Reports**. That panel now includes **Crash logs** and **View crash log**. Existing Copy summary, Find matches, Report issue and opted-in GitHub reporting work with sanitized crash metadata; raw stacks and dumps are NEVER uploaded automatically.
- Crash protection setup failure is visible in the status footer instead of silently pretending it is enabled.

**Caution:** Power loss or forced termination can also leave an unclean session marker; the diagnostic accurately reports an abnormal exit, not an unverified root cause. No trace from the user's earlier Phantom Crash crash was captured, so its original cause cannot be asserted.

## Duplicate JMP files — fixed in import and in existing library

- Full content SHA256 plus conservative JMP semantic equivalence detects patches with identical edits even if the author, description, BOM, whitespace or line endings differ.
- Import JMP, loose-file drag/drop and folder imports all skip existing duplicates. Exact message **Duplicate patch found! No need to add.** in a red bottom-log line; no redundant file or overwrite.
- Pre-existing duplicate JMP files are collapsed in the patch catalogue without deleting source or losing local comparison screenshots.
- Distinct patches with the same filename remain independent copies.

## Convert & Create

- IPS run-length-encoding checkbox appears only for IPS, EIPS and all-formats pack output.
- Encryption password appears only when creating encrypted IPS (including a pack) or decrypting a supplied EIPS source. Plain IPS needs no password. JMP/BPS/Xdelta output hides these controls unless a password is needed for encrypted input.

## Full JMP patch notes

- Replaces the formerly truncated single-line notes footer and literal HTML markup with a **READ FULL NOTES** viewer.
- Notes display with preserved line breaks from HTML br tags, basic paragraphs/list markup and escaped entities; the entire normal note body is word-wrapped and scrollable, with **Copy all notes**.
- The renderer does not execute scripts or remote HTML content.

## Independent list scrolling

- Mouse wheel scrolls only the patch list, selected patch's record/group list, or full notes—whichever is under the pointer. No simultaneous scrolling.
- Each list has a visible draggable/clickable scrollbar. At compact heights, the patch list shows fewer rows to preserve record space.

## Safety and validation

All original game and JMP files remain unchanged; the managed self-update v1 flags, filenames, SHA256 manifest and app-data directories remain compatible. No extra runtime/installer is introduced; no recurring GitHub Actions are added.

Source and remaining QA: docs/CRASH_RECOVERY_AND_UI.md and docs/STATUS.md. Go tests simulate clean/abrupt exit, panic, sanitized reporting, semantically duplicate JMPs, folder imports and text formatting. Native Windows UI smoke tests exercise the scrollbars, long notes and conditional conversion fields. Real Windows desktop mouse/file-picker testing, the original Phantom Crash XBE reproduction and actual Windows fatal-exception behavior remain field-QA items.
