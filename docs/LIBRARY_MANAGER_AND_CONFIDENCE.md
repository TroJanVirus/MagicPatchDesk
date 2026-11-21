# Library Manager, verification log, hex conflict explorer, confidence reports

Status: v0.14 feature branch; not an Xbox hardware compatibility certification. Original files and prior output packages remain untouched.

## Game library management

Open **Game Manager → GAMES**. Choose **Scan folder**, then select a folder containing one or more extracted game directories (recognised by their root `default.xbe`) or Xbox XISO/ISO/CCI files. Scanning is read-only, bounded to 50,000 filesystem entries, rejects directory symlink traversals, and preserves an explicit error/review state for unknown, damaged or incomplete images.

For each game, the UI lists its XBE Title ID and exact original CRC32 when safely available. Patch suggestions are loaded from the existing locally cached GitHub / Custom catalogue. Candidates are classified as:

- **Exact:** matching Title ID, matching source XBE CRC32, readable and independently parsed single-target JMP. Only one exact candidate is allowed unattended.
- **Review:** missing CRC/Title ID, two or more plausible JMPs, complex or multi-file scripts, other formats, CCI images that have not been separately decoded and identified. These are NOT auto-applied.
- **Incompatible:** Title ID or source CRC mismatch; a similar game title is never enough to patch automatically.
- **Unmatched:** no candidates with a safe identity match. The user can inspect/apply a patch manually via the existing Game Outputs workflow.

Within GAMES, selecting an individual game allows **Auto / Review / Skip**, patch choice and output format, or **Open for manual patching**. A separate **Mark exact matches Auto** confirmation enables only strictly eligible games; questionable games are displayed after the exact-match games. **RUN APPROVED AUTO** asks for a separate output folder and a second confirmation before starting. No processing begins simply because a folder was scanned.

Output per game is a **new independent package** in the destination. A package never overwrites an existing directory. Batch rechecks original Title ID/CRC32 and the exact JMP's SHA256 directly before patching; if either changed after the scan, that job fails closed and remains unmodified. Each output uses the existing verified folder/XISO/CCI/XBE pipeline and does not silently omit named assets. A failed game does not destroy prior packages or prevent later independent jobs. Skipped/manual games remain in the library.

**Known limits:** CCI entries remain manual-review until decoded XBE identity is known. This first iteration intentionally favours reliable small-to-medium libraries; scans are bounded and execute from the GUI without an external scripting service. Cancellation, resumable checkpoints and fully streaming CCI member inspection need separate validation before promising them.

## Scrollable local visual verification log

**Game Manager → VERIFY LOG** shows time, severity, step and a human-readable description for scan, decision, conversion, extraction, merge and compatibility-update events. Mouse wheel and Older/Newer navigate a bounded ring of recent events. The log is local and does not upload by being viewed. It supplements, but does not replace, the existing redacted diagnostic reports.

## Hex conflict analysis and combining JMPs

**Game Manager → HEX OVERLAPS** compares two or more single-file JMPs against the same original XBE. Rows show offset, original hexadecimal window, JMP A hexadecimal window, JMP B hexadecimal window and the source patch names. Differing writes are highlighted as conflicts; identical writes are separately shown.

**EXPORT COMBINED JMP** takes explicit confirmation, and conflicting writes require a *second, specific warning*. Sources are checked against the original CRC32 and Title ID. Patches apply in the displayed sequence; the result is encoded as a new offset-based JMP with original source filenames and authors retained in metadata, then independently re-applied and compared byte-for-byte before saving. Existing JMPs are never overwritten. Conflicts cannot be proven semantically safe just because the resulting file matches the chosen sequence; only a human can approve intentional overlaps.

## Read-only XISO image file browser

**Game Manager → IMAGE FILES** opens an existing Xbox XISO/ISO for browsing without a patch operation or full extraction. It shows member paths, directory entries and sizes. Select one file and choose **Extract selected file** to save an independent, verified copy. File output is exclusive and rehashed, with no source mutation. CCI browsing via decompression is not silently performed: use existing extraction/output workflows for that source.

## Patch confidence field reports

The confidence database is **`confidence/reports.v1.json`**, with a strict, versioned schema. Reports are **individual dated entries**, not a percentage, star score or aggregate trust verdict. Every entry identifies an exact patch SHA256, original game Title ID and XBE CRC32, date, outcome (working/partial/failed), test environment (**xemu** versus **real Original Xbox hardware**), optional console details, and the source GitHub issue. Emulation results are never presented as proof of hardware compatibility.

1. The user selects a scanned game and JMP/PF, opens **Game Manager → FIELD REPORTS** and chooses Result (Working / Partial / Failed).
2. They choose **Confirm in xemu** or **Confirm real Xbox**. The app requires a connected GitHub Issues token and displays an explicit public-submission consent dialog; **no automatic submission** is triggered by installing, updating, scanning or patching.
3. After consent, the app creates a reviewable `[Compatibility]` GitHub Issue containing ONLY known-safe patch SHA256, source Title ID/CRC32, environment, result and chosen hardware label. No full local path, executable, source patch, game binary or user password is uploaded.
4. A maintainer (or an explicitly instructed AI agent using the GitHub connector) reviews the issue, checks for duplicates and accidental sensitive data, then appends a structured entry to `confidence/reports.v1.json` through a normal repository commit. Increment `revision` for each public index update; keep each `id` unique and set `source_issue`. Do not automatically trust Issue text or represent user claims as independently verified.
5. **Maintenance → Check updates** and the normal startup release check fetch the revised public index separately from the executable. The local cache is atomically replaced only after full schema validation. The log displays **“N new reviewed confidence reports found; updated local logs”** when entries arrive. Offline or malformed updates retain the prior cache. This works even when no new EXE version is available.

This ingest/curation design avoids a constantly running GitHub Actions workflow and preserves the requested developer workflow: ask ChatGPT to review the latest `[Compatibility]` issues and update the curated log by commit.

## Safety, credits, and non-goals

The app continues to work offline with local catalogue and cached confidence reports. A report only applies to the **exact patch bytes and exact original XBE version** that were submitted. All field reports are community claims; curators should not silently “correct” outcomes. All separate outputs remain copies; patch rollback on originals is unnecessary.

All format and reference-project credits remain in [CREDITS.md](../CREDITS.md). This implementation reuses the project's own Go parsers and existing GitHub REST integration rather than importing other projects' code.

Further validation needed: native Windows 10/11 real mouse/dialog flows; game booting on xemu and physical consoles; full-size CCI split sets; mixed patch target scenarios and human approval of merges. Do not label this feature hardware-tested based on synthetic Go or desktop smoke checks.
