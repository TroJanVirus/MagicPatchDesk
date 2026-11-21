# Magic Patch Desk diagnostic reports (v0.8 developer preview)

## Goal

A failed patch, preview, PF operation, conversion, or patch pack should produce a self-contained report that can be given to the developer for investigation and future compatibility improvements. Normal application activity never needs to call home.

## What is recorded

Each report has a unique random ID, UTC time, application version, OS/architecture, operation stage, classified failure category, source type, input/output patch formats where known, valid Title ID and CRC32 metadata if already available, and applicable record/offset numbers. PF operations that report warnings can also produce a report with warning severity and warning count.

The report **does not copy**: original Xbox game bytes, patch payload bytes, search patterns, passwords, full local or network paths, usernames, arbitrary patch comments or unchecked error text. A failure explanation is selected from a fixed list of categories. Unknown errors are classified as `operation_failed`; their raw text remains visible on the computer's status bar, not in the report. Source file names are represented by their extension.

## Storage

Reports are saved as individual private JSON files under `%APPDATA%\MagicPatchDesk\Reports\failure-<ID>.json` (the application's configured data directory followed by `Reports`). They are never automatically uploaded. Files are never overwritten, and users can remove them from the folder at any time.

## Windows interface

Open **Settings > Problem Reports**:

- **View last report**: display a safe summary of the most recent report; reports from a prior session remain available in the Reports folder.
- **Copy summary**: put the safe summary on the Windows clipboard for support chats.
- **Open reports folder**: inspect the full JSON documents before sharing them.
- **Report to developer**: after confirming a warning that issues can be public, open a prefilled GitHub issue at `TroJanVirus/MagicPatchDesk`. Nothing is uploaded until the user manually submits it.

For the most effective investigation, include *manual reproduction steps*, whether the problem happened with loose files, an XISO, or a full disc ISO, and whether the game still boots. Uploading copyrighted games or passwords is neither necessary nor recommended.

## Verification

`go test ./...` exercises report generation, private local writing, duplicate prevention, failure categorization, warning categorization, recent-report retrieval and intentional exclusion of sensitive strings. Windows x64 compilation is required. The GUI, clipboard and browser workflow still require hands-on Windows testing.

## v0.9: grouped reports

Reports now also carry `group_id`, derived deterministically from the validated Title ID, stable failure category and affected operation. Older v0.8 JSON reports remain readable and are grouped at display time. Settings presents the most recent groups with counts, Next/Previous navigation, a safe summary, and a **Find matches** action that searches GitHub issues for the group ID. **Report issue** searches first or, after an explicit choice, opens a prefilled new issue with that ID in the title. See [DIAGNOSTIC_GROUPING.md](DIAGNOSTIC_GROUPING.md). No automatic telemetry or GitHub Actions runs are involved.
