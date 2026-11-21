# Recent files and quick navigation (v0.11)

## Purpose

The native Windows application has a **Recent** sidebar workspace for returning to commonly used game inputs, custom patches and generated output copies. It does not open a previously saved output as an original XBE automatically, because that can lead to accidentally re-patching a modified game.

Supported categories:

| Category | Selecting the entry |
| --- | --- |
| Game source | Loads the original XBE / supported ISO in Patch Library |
| Patch file | Loads the standalone patch in Patch Library |
| Conversion source | Populates the stock/original field in Convert & Create |
| Conversion patch | Populates the conversion patch field, retaining JMP/PF mode detection |
| Modified XBE | Populates the modified XBE field in Convert & Create |
| Saved output | Opens the containing folder, not the output as a new source |

The workspace has **Open selected**, **Open folder**, **Copy path**, **Forget**, **Clear history**, Previous and Next. Missing files are clearly reported when opened. Forget and Clear History never delete game files, patches, output copies or diagnostic reports.

## Keyboard navigation

- **Ctrl+O** — choose an original game input.
- **Ctrl+P** — choose a standalone patch.
- **Ctrl+F** — go to Patch Library and focus search.
- **Ctrl+R** — open Recent Files.
- **Ctrl+1** through **Ctrl+6** — navigate Patch Library, Convert & Create, Settings, Diagnostics, About and Recent respectively.
- **F5** — rescan the local patch library (not a GitHub synchronization).
- **Esc** — leave text field focus.

Keyboard shortcuts are ignored while an active file modification is in progress; F5 is reserved for local indexing. The original patch confirmation, checksum and source requirements remain unchanged.

## Local-only storage and deletion

The app stores at most **18** unique file paths with category and last-used UTC time in a local `%APPDATA%\MagicPatchDesk\recent-files.json` (or the configured application-data root). This intentionally includes full local file paths *on that PC only* so the history can reopen the right file. Do not attach this file to public bug reports or GitHub issues. Nothing from this history is sent to the diagnostics service, PatchHub or GitHub. This is a convenience feature, not a cloud sync.

Updates are written to a temporary file and renamed so a failed write does not normally destroy existing history. Unrecognised entries and malformed metadata are excluded on load. **Clear history** deletes only this local history file after confirmation; it does not remove game assets or other preferences.

## Validation limitations

Go tests exercise history ordering, deduplication, capacity, persistence and invalid data. The native Windows x64 program compiles. Windows UI and physical Xbox title testing must still be done manually; do not claim they passed based on cross-compilation alone.
