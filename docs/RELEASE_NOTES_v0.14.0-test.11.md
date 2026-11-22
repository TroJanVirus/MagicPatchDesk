# Magic Patch Desk v0.14.0-test.11 — Readable labels and clearer output choices

This preview focuses on the Windows interface. No patch format, game compatibility, private data, or update-protocol changes.

## Interface improvements

- Buttons now **measure their text in Segoe UI** before drawing it. They use a slightly smaller caption font if necessary and, where space permits, wrap rather than cutting off words.
- **Maintenance → Update settings** has properly sized, readable controls: `Previews on`, `Check + sync at start`, `App folder` and `Data folder`. At narrower window widths, folder actions move to a separate row instead of being squeezed to `App fol…` and `Data fol…`.
- Screen headings now show literal **ampersands**: `Discover & patch` and `Updates & maintenance`, instead of accidentally displaying the Windows mnemonic underline.
- **Patch Library** uses more of the available width for game titles and lets long game titles and patch names wrap. The `Widescreen` category badge is sized using the real font rather than an inaccurate fixed character width.
- The lower Patch Library output prompt now says **`OUTPUTS: Choose the formats to create from your selected game and patch`**. It explains what to do instead of misleadingly saying `Choose a source game`. The guidance can use two lines when needed, while the output buttons remain separate.
- Header titles and subtitles may wrap within their allocated area where required.

## Verification and compatibility

The existing Windows smoke checks were extended to assert that the four Maintenance update-setting actions are present, their captions fit the real Win32 click areas, and their hit targets do not overlap at **1024px and 800px** window widths. The output hint is also checked. These smoke assertions are compiled into the EXE but cannot run on a Linux build runner; hands-on native Windows visual QA is still required.

The owner-provided end-user `README.md` was **not changed**. Updater protocol v1 and the canonical `MagicPatchDesk-Windows-x64.exe` / `SHA256SUMS.txt` release assets are unchanged. All previous versions remain available for rollback.

The existing unexpected-exit issue is **not** claimed resolved by this UI-only change.
