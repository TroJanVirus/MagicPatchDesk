# Magic Patch Desk v0.14.0-test.5 — One patching workflow and reliable portable updates

**Developer prerelease; unsigned Windows 10/11 x64 EXE.** Continue to treat retail Xbox boots and real desktop UI interaction as field QA rather than automated proof.

## Six requested changes

1. **Game Outputs is no longer a separate sidebar tab.** Load an original game file or extracted game directory directly in Patch Library, select a patch, click **CHOOSE OUTPUTS** and select any combination of independently verified Patched XBE, Extracted folder, Game XISO and Cerbios CCI. Full-disc ISO is only offered for a real full-disc input. **Preserve current format** is an explicit option that can be combined with others. The **APPLY PATCH** button is visibly disabled and has no click target until a source, patch and at least one supported format are explicitly selected. Choosing another source resets the selection. The existing safe game-export and format-specific verification engine remains intact.
2. **Game Manager is now labelled Batch Process Library**, preserving scanning, review queue, hex comparisons, verification log, individual field reports and supported batch operations.
3. **Settings and About appear at the bottom** of the sidebar, including compact Windows screens; nonessential collection metadata may be hidden to prevent overlap.
4. Maintenance's **Self Update** section is now **Check for Updates**. The former Channel & Storage controls are nested inside this section as **Update Settings**, with visible **Previews ON = allow downloading preview builds** text.
5. **Automated Diagnostic Reports** shows only **Connect** when no GitHub credential is saved. After connection, **Disconnect**, **Auto-report ON/OFF**, and **Sync now** become available. Windows Credential Manager stores the token; no PAT goes into the UI, config files or error logs.
6. **Portable updater block removed.** The old message "an installed copy exists; open it to update" occurred because portable updates were hardcoded to refuse an existing managed installation, not because both EXEs had the same name. The updater now updates **the EXE you launched**, without touching a separately installed or separately portable copy. Both retain independent prior-version rollbacks. A locally authorized portable update intent validates the target and expected SHA256 before using the unchanged v1 helper command and restarting the updated portable EXE.

## Backward compatibility and migration

Canonical download: `MagicPatchDesk-Windows-x64.exe`, accompanied by `SHA256SUMS.txt`. Update protocol stays **v1**, with `--mpd-apply-update-v1` and `--mpd-healthcheck-v1` retained. Installed program path and `%APPDATA%\MagicPatchDesk` data remain unchanged; no games, patches or user reports are deleted. No additional runtime is needed. If you are still running an **older portable build that already hits the updater blocker**, manually download and run **this** version once. Older compiled code cannot be fixed remotely; subsequent portable updates use the new implementation.

A portable folder without write permissions can still prevent in-place updating, but the app reports that failure instead of silently modifying another installation.

## Test gates and remaining QA

Automated tests assert format choice is required, preserve-source behavior, multi-selection and source-specific impossibility checks; mismatched/tampered portable update intent/checksum/parent rejection; separate portable/managed rollback paths; and native Windows 1024×720 UI visibility for the integrated chooser, disabled Apply, Settings/About placement, batch rename and credential-gated diagnostic buttons. Full Go tests, race/vet, Windows compile, EXE checksum and exact executable updater/UI healthchecks must pass before publication.

**Not yet user-tested:** physical Windows portable replacement from an older live EXE and real click/drag behaviour, XISO/CCI export on representative retail games, or original Xbox boot compatibility. The verified release gate confirms the new code builds and its synthetic flows execute, not a complete hardware validation. Source and migration details: [Patch Library outputs and updates](https://github.com/TroJanVirus/MagicPatchDesk/blob/main/docs/PATCH_LIBRARY_OUTPUTS_AND_UPDATES.md).
