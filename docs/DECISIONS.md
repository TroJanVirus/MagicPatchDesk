# Project owner decisions — MagicPatchDesk

**Purpose:** This is the durable record of **explicit user decisions**, as opposed to speculative feature ideas. It is intended to survive new ChatGPT conversations. Always reconcile a new instruction from the user with this document; **the newest explicit decision wins**. For present code/release facts, use [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) and [STATUS.md](STATUS.md), not this decisions log.

## Core product / workflow

| Decision | Requirement | Status / qualification |
|---|---|---|
| Native app, offline first | Actual Windows 10/11 x64 Win32 GUI, Go core, portable or optional managed install; no dependence on web-only UI. Imported local patches work without a network. | Implemented; native UI click-through QA pending |
| Never destroy or overwrite originals | Create a **separate copy of every patched file** and independent new game output package. User does **not** want a complicated undo/rollback workflow for game patches; updater's own previous-EXE fallback is a separate mandatory protection. | Core product rule |
| Explicit output formats | An XISO example/input must not force XISO output. Users choose extracted folder, standalone patched XBE, rebuilt game-partition XISO, CCI, or valid full-disc ISO where explicitly safe. A supplied full-disc ISO is not automatically required as output. | Implemented with source/layout constraints; real hardware QA pending |
| Patch artifact redundancy | JMP can be converted into all representable standard patch variants, optionally bundled into source-verified `.mpdpack`; each named target stays separate. Do not silently drop records or fake whole-game output from a loose XBE. | Implemented subset; test per format |
| Visual integrity feedback | Present verification through a **scrollable visual log** containing successes/warnings/errors rather than a large separate preflight workflow. | Implemented |
| Library scanning and batch management | Scan a folder containing existing extracted / image games, show **patches available per game**, let a user approve or skip individually, or opt into **fully automatic processing for strictly matched cases**. Put unusual/complex games at the end for **manual approval**; do not auto-guess versions or silently patch ambiguous games. | Conservative implementation; complex/CCI images remain review-first |
| Combine JMPs with conflict visibility | Detect overlapping byte changes; show a **visual hexadecimal comparison** and warn the user about conflicting writes; require explicit approval before combining conflicting patches. Preserve source attribution and original files. | Implemented; semantic patch compatibility remains a human judgement |
| Image browser is useful without patching | Allow read-only exploration and selective extraction of game-image contents **before** deciding to patch. | Implemented for supported XDVDFS images |
| Minimise friction | Fast, clear native GUI with per-game choices; exact matching and automated success verification where safe, not popups for routine safe operations. | Ongoing usability priority |

## Community reports and automated triage

| Decision | Requirement |
|---|---|
| Compatibility is **not a score** | No percentage rating or star rating. Display each individual report as a dated, inspectable **log** with patch identity, result and test details. |
| Report result | Record **working / partially working / failed** for the **exact patch SHA256 and original game XBE Title ID + CRC32**, not a vague title match. |
| Environment matters | **xemu/emulation** and **real Original Xbox** must be distinguishable. Emulation evidence never implies real-hardware compatibility. |
| Xbox motherboard revision | Real-console reports offer **v1.0, v1.1, v1.2, v1.3, v1.4, v1.6, v1.6b**, plus **I don't know**. Never guess based on the game or the user's PC. |
| RAM and CPU modifications | For **real hardware**, record RAM upgrade status and CPU upgrade status **independently**: **No upgrade / Upgraded / I don't know** for each. Do not assume stock and do not attach real Xbox revision/upgrade information to xemu-only reports. |
| Prior compatibility evidence | Old reports without these fields stay readable as **Not recorded**, distinct from a user choosing **I don't know**. No retroactive invented hardware specifications. |
| Sending reports | User should easily confirm a patch as working/partial/failing, with a clear consent step before submitting a **public** GitHub Issue. Submitted reports are **claims, not independently verified facts**. Never upload game bytes, raw patch payloads, local paths or credentials. |
| Database and refresh | Maintain a reviewed compatibility report index in the repo at `confidence/reports.v1.json`. A maintainer or explicitly tasked AI agent reads/validates new GitHub Issues and commits curated entries. App update checks should also download newly reviewed confidence reports and announce new entries even when the EXE is up-to-date. No numerical aggregate. |
| Problem reports | Diagnostic problems should be locally recorded, grouped and privacy-filtered; user may ask ChatGPT **"check latest reports in repo for problems"** and have the connected GitHub app inspect issues directly, without repeated manual report uploads. Automatic diagnostic issue reporting is optional and must respect user opt-in and privacy. |

## Maintenance, releases and project sustainability

- **Every future released version must update from an older installed app** through the established v1 update protocol. Keep the canonical Windows EXE name, SHA256 manifest, update helper flags, app-data directories and rollback-to-previous-EXE support. The design is in `docs/UPDATE_CONTRACT.md`.
- Releases should be **GitHub Releases**, with direct, literal URLs to the EXE and checksum rather than distributing arbitrary ZIP builds through chat. Do not call an unsigned developer preview a stable or console-certified release.
- **Do not burn GitHub Actions usage** unnecessarily: the user has previously been near the account's usage limit. Avoid recurring workflows, frequent reruns and one-shot release publisher files left behind. Prefer a single gated verification when justified.
- **Attribute reference work properly.** `CREDITS.md` and the app's About view must acknowledge references, origin, source links, licensing and the difference between format/workflow inspiration and any legally reused implementation. Repackinator / Team Resurgent is a workflow reference, not code silently incorporated into this project.
- **New conversations must recover context from the repo alone**, not from ChatGPT's conversation history. Keep `AGENTS.md`, `PROJECT_CONTEXT.md`, `docs/STATUS.md`, this decisions file and dated `AI_HANDOFF.md`/`docs/PROJECT_STATE.md` aligned after meaningful work.

## Explicitly deferred or rejected for now

- **No FTP transfer/deployment workflow yet.**
- **No automated launching of xemu yet.** Using xemu as *a field-report environment* is fine; building an emulator-launch feature is not requested.
- **No new headless command-line user interface yet.** Internal healthcheck/updater flags remain mandatory.
- **No patch undo workflow** replacing copy-on-write semantics.
- **No unrequested format expansion** or additional image output types merely for theoretical coverage.
- **No speculative percentage confidence ranking**, and no interpreting successful compilation as confirmation a game boots.
- **No automatic upload of user data** while scanning libraries, verifying output, updating the app or browsing confidence history.

## How to update this decisions file

Only record a decision here when the **owner actually made it** or explicitly accepted it; ideas from assistants remain proposals until approved. When a decision is superseded, edit the active row and add a brief dated note with the old/new behavior; do not leave contradictory active instructions. Also update `PROJECT_CONTEXT.md` and the task-specific documentation. This file is not a bug tracker, development TODO list or a claim that an idea has already shipped.

## 2026-11-22 — additional explicit product decisions

- **Import JMP duplicate behaviour:** if an identical patch already exists anywhere in the indexed library (including GitHub-synced patches), importing it is a **no-op** and shows the red bottom-log message **“Duplicate patch found! No need to add.”** Do not overwrite a same-named *different* patch.
- **PatchHub screenshots:** show before-and-after comparison images for the selected patch **in the Patch Library**, not just a game cover. A patch may have **several** screenshots; the user must be able to browse all discoverable images. Be accurate about which patch/version each screenshot belongs to, and retain offline functionality where cached or imported images exist.
