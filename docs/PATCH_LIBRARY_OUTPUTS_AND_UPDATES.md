# Patch Library output workflow, navigation and portable updates

**v0.14.0-test.5 implementation and future-agent handoff.** This document supersedes the separate Game Outputs tab description from older release notes. Treat the repo's current source and verified release as authoritative.

## 1. One single-game workflow — inside Patch Library

The old **Game Outputs** sidebar entry was redundant. Patching a single game now happens entirely in **Patch Library**:

1. Select a catalogue game and one JMP/PF (or load a supported direct patch). Use **Open file** to choose an original XBE, full-disc ISO, game-partition XISO, or Cerbios CCI. Or **Open folder** and select the extracted game's **root default.xbe**; the program processes its containing complete game directory.
2. The **APPLY PATCH** button is *inert, visually disabled and has no hit target* until all three things are true: a source game is selected, a patch is selected, and one or more compatible output formats are **explicitly chosen**. Do not automatically apply a previous source's output preference.
3. Click **CHOOSE OUTPUTS** to open the selection overlay. Independently check one or more of **Patched XBE**, **Extracted folder**, **Game XISO**, **Cerbios CCI**, or **Full-disc ISO**. Also check **Preserve current format** to include an output matching the source's type. Preserve is a first-class explicit choice, not an implicit default; combining it with additional formats forms a union without duplicate output artifacts.
4. Format availability depends on actual input identity: a loose XBE can only produce XBE, full-disc ISO can only be produced from a real full-disc ISO, and other complete-game inputs may produce extracted/XISO/CCI/XBE. A PF which touches other game members cannot be represented by XBE-only or full-disc-ISO output. Unsupported conversions are disabled and again checked on application.
5. Preview remains accessible before applying. The existing `core.ExportGameOutputs` stages all chosen outputs under a **new, independently named package folder**, verifies game and patch identity and never overwrites the source or a previous output. Errors and verification events remain in the local log and diagnostics.

All formats need only one explicit selection per chosen source. Choosing another source **clears** output choices and requires deliberate selection again. Stored historical per-kind preferences remain readable for backwards compatibility but are not silently applied to the new source.

**Code:** `core/output_choices.go`, `cmd/magicdesk/game_outputs_windows.go`, `cmd/magicdesk/library_outputs_windows.go`, `cmd/magicdesk/main_windows.go`, `cmd/magicdesk/paint_windows.go`. The old Game Outputs renderer may remain in source for backwards compatibility with older code/tests, but there must be **no separate visible sidebar tab or action** accessing it. Batch library processing remains separate; `core.ExportGameOutputs` and its no-overwrite source integrity rules are unchanged.

## 2. Sidebar / Maintenance changes

Top workspace navigation: **Patch Library**, **Convert & Create**, **Batch Process Library** (formerly Game Manager), **Diagnostics**, **Recent**, **Maintenance**. **Settings** and **About** are permanently anchored to the bottom of the sidebar, including at a 720p-height window. The compact window can omit the nonessential collection-count card rather than cover navigation.

Maintenance now contains one combined **CHECK FOR UPDATES** section. It houses both the existing Install/Check/Update controls and an **UPDATE SETTINGS** subsection (formerly the separate Channel & Storage section). A readable caption explains **PREVIEWS ON = allow downloading preview builds**; automatic release checks and App/Data folder controls also live there.

The **AUTOMATED DIAGNOSTIC REPORTS** section shows **Connect** and no other action when there is no valid GitHub Issues credential. Once connected, it hides Connect and shows **Disconnect**, **Auto-report ON/OFF**, and **Sync now**. The connection status comes from Windows Credential Manager once at startup and is updated immediately after successful connect/disconnect; no token is retained in UI state, printed or saved to app data. Auto-report stays opt-in.

## 3. Portable updater no longer blocked by a separate installation

The earlier error **"an installed copy exists; open it to update"** was due to an explicit branch in `maintenance_windows.go` refusing portable app updates whenever `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe` existed. The canonical GitHub download filename was **not the cause**.

From v0.14.0-test.5 onward, **Update now updates whichever EXE the user launched**:

- Managed installation: unchanged destination `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe`, unchanged stable `--mpd-apply-update-v1` helper protocol, `MagicPatchDesk.previous.exe` backup and v1 checksum validation.
- Portable: stage and SHA256-verify the new EXE beside the running portable binary, run the staged EXE's `--mpd-healthcheck-v1`, and create a bounded, user-initiated local update-intent file binding the **current PID**, original target path and expected checksum. Make or refresh a co-located `MagicPatchDeskUpdater.exe` helper from the *running* compatible app, to avoid invoking an older helper that only understands the installed destination. The helper verifies the exact intent, staged digest and folder before replacing the now-exited portable EXE, retains the previous version under `<original-name>.previous.exe`, then relaunches the updated **portable** copy. A separately installed copy remains untouched.
- No spontaneous copy from the portable folder into the installed folder and no ambiguous "installed copy already exists" refusal. The app clearly identifies which copy it is updating. A read-only portable folder or missing permissions produces an error without overwriting the old executable.
- The public release continues to use the canonical asset `MagicPatchDesk-Windows-x64.exe` plus `SHA256SUMS.txt`. The `UpdateProtocol = 1` flag and existing 5-argument internal v1 helper are unchanged. No remote unverified executable is ever installed.

**Migration caveat:** previous EXEs cannot retroactively gain the new portable-update logic. If an older *portable* copy already hits this bug, the user must **manually download and run v0.14.0-test.5 once** (or update via their already-installed copy), then later portable updates can proceed through the fixed protocol. Neither the portable nor installed updater should silently replace the other's EXE.

**Code:** `core/update_intent.go`, `cmd/magicdesk/maintenance_windows.go`, `docs/UPDATE_CONTRACT.md`. Tests cover explicit source-format choices, multiple selected outputs, impossible conversions, portable update intent/path/digest tampering, managed-vs-portable rollback names, and native GUI hit-test state at 1024×720.

## Release validation / limits

Before claiming a downloadable working version: run full Go tests, race detector, vet, Windows x64 compile, SHA256 manifest check, native Windows unit tests, **the actual release EXE** `--mpd-healthcheck-v1` and `--mpd-ui-smoke-v1`; only then publish a prerelease and remove the one-shot release workflow to save GitHub Actions minutes. A green UI smoke does **not** prove human clicking, real ISO processing or a live Windows updater swap. Actual Windows 10/11 update-from-old-portable and game-image tests remain manual QA. Keep PROJECT_CONTEXT.md and docs/STATUS.md truthful and self-contained for future new conversations.
