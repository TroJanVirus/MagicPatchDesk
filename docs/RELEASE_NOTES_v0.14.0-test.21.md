# Magic Patch Desk v0.14.0-test.21 — cover download recovery and native draw check

## Why test.20 was insufficient

The user installed test.20 and synced GitHub but American Idol (Prototype) still showed a letter tile. Live diagnostic GitHub Actions run https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45142481317 confirmed:

- The exact PatchHub URL `All-Patches/titles/American Idol (Prototype)/434D0027.jpg` exists.
- `core.DownloadCover` successfully downloaded and validated the **154,187-byte** JPEG.
- `core.SyncPatchHub` and `core.LoadCatalog` recovered the correct game name, folder, source URL and Title ID `434D0027`.

No website, JPG filename or JMP change is necessary. The remaining failure was within native Windows cover requesting/painting or local state; this candidate addresses identifiable flaws in those paths.

## Changes since test.20

- Replace paint-driven three-slot drop-on-full cover scheduling with **three persistent cover workers and a bounded 48-job queue**.
- Always notify the native UI after worker completion, even when a request fails, so pending visible covers get another attempt.
- Use a separate **18-second timeout per candidate** rather than one shared timeout; a broken canonical URL cannot consume all time available for a valid fallback.
- Reduce missing-cover retry cooldown to 2 minutes, while preserving memory limits and offline disk cache.
- If a local cover filename is stale, invalid or undrawable, still try verified PatchHub cover paths.
- Add `smokeVerifiedCoverDrawing` to the real Windows `--mpd-ui-smoke-v1` suite. It writes a short temporary JPEG, checks the exact hashed URL cache path, validates it with the normal decoder and actually renders it via GDI+, then removes the temp fixture.

## Compatibility and release gate

Same `MagicPatchDesk-Windows-x64.exe`, `SHA256SUMS.txt`, v1 updater, portable and installed update paths, embedded JMP icon, no changes to executable patch applicability, privacy, original game files or human README. The one-shot Actions run https://github.com/TroJanVirus/MagicPatchDesk/actions/runs/45143154661 **passed** portable Go tests/race/vet, media fixture, updater checks, Windows PE build and checksum, native Windows GUI smoke including actual image drawing, and healthcheck **before** publishing the prerelease. No new always-on workflows.

**Still needs owner field test:** Verify live cover download and display on Windows; synthetic Windows GDI+ drawing plus a successful Linux live HTTP probe do not prove the user's precise environment works.

