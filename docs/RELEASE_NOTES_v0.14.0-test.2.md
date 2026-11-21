# Magic Patch Desk v0.14.0-test.2 — Detailed physical Xbox field reports (PREVIEW)

A focused update to the Game Manager's **FIELD REPORTS** workspace. Keeps every original game file and source JMP unchanged, maintains the version-agnostic update protocol and retains independent xemu vs physical-Xbox results.

## What changed

- **Xbox motherboard revision** can now be specified for a real-hardware test: **1.0, 1.1, 1.2, 1.3, 1.4, 1.6, 1.6b**, or **I don't know**.
- **RAM upgrade status** is independently selectable as **No upgrade**, **Upgraded**, or **I don't know**.
- **CPU upgrade status** is independently selectable using the same three choices.
- The three choices appear in the public GitHub Issue confirmation dialog, in its structured issue content, and in the detailed compatibility history for that exact game/patch combination.
- Reports for **xemu** omit these physical-hardware fields entirely. Results continue to be individual dated entries, never percentage scores.
- **Legacy field reports** without hardware details remain readable, displayed as **Not recorded** rather than falsely inferred stock hardware or a user's explicit unknown answer.
- All newly submitted real-hardware reports include all three fields with an explicit unknown available. Wrong/partial values fail validation. Changing selected games resets choices to unknown instead of guessing stock hardware.
- The original Xbox revision list is referenced from ConsoleMods community documentation, with appropriate attribution in CREDITS.md. Nothing from upstream code has been copied.

## Update and privacy guarantees

Standard `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` assets, `--mpd-apply-update-v1` and `--mpd-healthcheck-v1`, Windows app/data directories and previous-EXE fallback are unchanged. The public compatibility database remains a curator-reviewed JSON index; personal reports are not automatically trusted or uploaded. Hardware fields are supplied voluntarily and remain self-reported, not device-detected.

## Test coverage and limitations

The release pipeline requires complete Linux/Windows Go test suites, race detector, vet, Windows binary build, native update/healthcheck and headless Win32 GUI smoke at 1024x720 with unclipped revision/RAM/CPU controls, followed by SHA256 verification. Xbox/Cerbios game bootability and physical Windows mouse/clipboard dialogs still need manual verification.
