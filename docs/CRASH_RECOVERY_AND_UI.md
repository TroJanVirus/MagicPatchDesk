# Crash recovery, JMP notes and Patch Library UI repairs

**Scope:** v0.14.0-test.4 developer preview. This document records the defects reported after v0.14.0-test.3 and the implementation, limitations and next actions for future AI agents.

## Why the previous crash produced no report

The old reportFailure()/recordDiagnostic() path only captured handled errors and selected notification strings. It wrote privacy-filtered JSON asynchronously after the application had already handled the error. A fatal Go panic, native Windows exception, process kill or loss of power could terminate MagicPatchDesk before that routine ran. No persistent crash marker or startup recovery existed. The crash therefore left nothing in the app's existing Reports view.

**The exact root cause of the Phantom Crash XBE-selection crash remains unknown** without a runtime trace. Duplicate catalogue entries and concurrent selection are relevant reproduction conditions but not proof of the underlying fault.

## Durable crash evidence and app recovery

The app now creates a small session marker when it starts. It removes it on clean exit. When a subsequent launch finds a marker from a terminated process, it creates a human-readable private evidence log and a separate standard sanitized diagnostic JSON. Markers for still-running app instances are ignored. Each marker records a safe stage (for example read_xbe, auto_match, parse_patch), never an original game path.

- A caught panic in the Windows UI callback, main application or selected XBE worker synchronously writes its reason and stack to a private log and saves a matching diagnostic report before termination.
- An unhandled fatal runtime error, forced exit or other unclean termination is detected on the next launch. The generated report accurately says **unexpected exit** rather than claiming proof of a particular crash. Best-effort redirected stderr may contain a useful runtime trace.
- An unread crash count appears as a red footer notification until clicked; that click opens Settings / Problem Reports, while leaving every report and log on disk.
- Settings / Problem Reports has **Crash logs**, **View crash log**, **Copy summary**, **Find matches** and **Report issue** flows. The existing optional diagnostic sync can submit only sanitized report metadata.
- If crash storage cannot start, the status footer shows a red warning. Source, patches and saved game images remain untouched.

Paths (private, persistent user data):

    %APPDATA%\MagicPatchDesk\Reports\
      failure-<report-id>.json      privacy-filtered shareable summary
      Crash\
        session-<id>.json          removed on normal exit
        notice-<id>                retained until user reviews reports
        crash-<id>.log             private panic trace or unexpected-exit note
        runtime-<id>.log           redirected runtime stderr, if produced

**Privacy guarantee:** raw stacks, runtime stderr and native dumps might include game filenames, users' folders or private memory. They are NEVER automatically uploaded. GitHub Issues and opted-in automatic reporting receive only allowlisted category/stage/version metadata. The user may inspect a log and manually attach a personally reviewed copy. A log without a stack is still useful for identifying the last action, but cannot determine the exact exception.

**Optional Windows native minidumps:** Microsoft documents WER LocalDumps at HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\Windows Error Reporting\LocalDumps\<exe>, which requires administrator privileges. MagicPatchDesk deliberately does not silently change that registry configuration. See https://learn.microsoft.com/en-us/windows/win32/wer/collecting-user-mode-dumps for opt-in configuration. No process can promise a full stack after arbitrary corruption, power loss or termination; the session marker is the guaranteed *attempt* at next-launch evidence when the app data folder is writable.

## Duplicate patch handling

The prior single-file guard compared byte-for-byte data only. Even behaviorally identical JMP patches could pass through when the author, notes, line endings, or BOM changed. Folder import bypassed that check, and older duplicated files were still shown separately after import.

- Use exact SHA256 comparison first, then normalized JMP text and a conservative record fingerprint: source version/Title ID, CRCs, game/region, ordered edit bytes, target filenames, enabled/disabled state and group semantics. Different actual patches remain separate.
- Apply this check to **Import JMP, drag/drop and folder imports**; a duplicate is skipped without overwriting source files. Local screenshots in imported folders remain available.
- The Patch Library catalogue collapses duplicates *for display* even if they already exist on disk. It prefers the upstream GitHub JMP and retains any locally associated comparison screenshots. No existing JMP files are deleted.
- The red footer message is exactly **Duplicate patch found! No need to add.** with a verification log entry.
- Same-named **different** patches are copied with independent numbered names, never overwritten.

Tests include altered author, notes, Windows line endings and BOM, different patch byte edits, source region mismatches, folder-only duplicates and preexisting catalogue duplicates.

## Conditional IPS conversion controls

- **IPS RLE checkbox**: appears for IPS, encrypted IPS (EIPS), or an all-formats pack containing IPS.
- **Password**: appears for encrypted IPS output, optional EIPS in an all-formats pack, OR encrypted IPS selected as a *source* (decryption is still required even if the output is JMP/BPS/Xdelta).
- Ordinary unencrypted IPS has no password; JMP, BPS and Xdelta output have neither control unless decrypting EIPS input.

## JMP notes fully readable

The old footer rendered a single short string with an ellipsis, literally showing HTML fragments such as Translation Mod.<br><br>1. Apply patch to default.xbe. Now a clear **READ FULL NOTES** button opens a word-wrapped, scrollable native full-note viewer with **Copy all notes**. The formatter preserves HTML <br> line breaks, paragraphs, simple lists and entities, but never executes HTML, scripts or remote content. No notes are silently cut off for ordinary sizes; exceptionally huge malformed headers receive an explicit viewer safety-limit message.

## Independent scrolling + visible scrollbars

The Patch Library patch list and the selected patch's record/group list each have separate viewport bounds, scroll offsets and a visible draggable scrollbar. Wheel events follow **where the mouse is actually hovering**, so moving records never also moves the patch list. The notes overlay scrolls independently. Compact windows reduce the patch-list row count to reserve room for records. Native Windows UI smoke validates the buttons and offsets at a 720p-height window.

## Verification and remaining QA

Source: core/crash_guard.go, cmd/magicdesk/crash_windows.go, core/jmp_equivalence.go, core/catalog.go, core/jmp_notes.go, cmd/magicdesk/notes_scroll_windows.go, cmd/magicdesk/main_windows.go, cmd/magicdesk/paint_windows.go, cmd/magicdesk/diagnostics_windows.go. Automated tests: core/crash_dedup_notes_test.go and cmd/magicdesk/ui_smoke_windows.go.

Actual Windows 10/11 mouse and file dialogs, selection of the original Phantom Crash XBE, abnormal OS termination evidence and any real hardware boot must be verified by a user. A passing Go/Windows CI smoke is not an Xbox compatibility certification. If the app crashes again, reopen it and use Settings → Problem Reports to inspect/review/upload its *sanitized* diagnostic issue; do not upload copyrighted games or unreviewed raw dumps.
