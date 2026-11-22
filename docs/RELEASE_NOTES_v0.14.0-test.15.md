# Magic Patch Desk v0.14.0-test.15 — XBE picker crash-risk mitigation

A Windows user reported an abrupt exit **immediately after selecting the original XBE for a JMP that was already selected in Patch Library**. The supplied `v0.14.0-test.13` files show an abandoned session, but the private runtime fault log and a WER minidump were **not** supplied; the marker alone cannot prove root cause.

## What we changed

- **Background file inspection no longer writes the live UI model.** The original-XBE reader now returns an immutable file inspection result through a Win32 `WM_APP+4` message. The UI thread alone commits the XBE title, Title ID, whole-file CRC32, inspection errors, game match and patch selection. Earlier code wrote `app.inputInfo`, `app.inputPartSummary` and status from a worker while the native paint path read those same values without consistent synchronization; that was a concrete data race and possible native UI stability risk.
- **Stale file inspections are ignored.** Each source picker action receives a monotonically increasing generation; an old worker cannot replace the selected image or game if the user chooses another file before the first scan completes.
- **An explicitly selected patch stays selected.** If its game matches the XBE, automatic game identification no longer needlessly reparses and reselects the same JMP or triggers another image-discovery request. Custom record/group enabled states are retained. The UI reports whether the selected patch's expected CRC32 matches the chosen XBE instead of silently replacing the patch with a different candidate. A missing library match no longer discards the user's existing selection.
- **Loose-XBE safety checks** reject nonregular files, undersized images and executables exceeding 512 MiB before allocating space for the file.
- **More precise crash markers** distinguish file selection / in-progress inspection / UI result commit from generic `ui_callback`. The old build reset to `ui_callback` before the background load finished. These markers still do **not** supply an exception code or stack trace; a private `runtime-*.log` or Windows Error Reporting dump is needed for fault-level diagnosis.

## Regression coverage

The Windows GUI smoke tests now simulate **selected patch → matching XBE**, assert that the same in-memory JMP and toggled record states survive, and check that late worker results cannot change the active game. The existing source-file/CCI/ISO tests and full patch converter tests remain unchanged. Go tests, race detector, vet, Windows x64 cross-build and the updater contract are required release gates.

**Honest limitation:** the user's crash could have another cause, including a native Windows fault. The supplied recovery markers cannot establish whether this concurrency fix resolves it. The Win32 smoke test is compiled on Linux but cannot execute there; hardware and real-file testing are still needed.

Your original game and JMP files remain read-only, and nothing is automatically patched. Human-written `README.md` is deliberately unchanged. Version is `v0.14.0-test.15`; updater protocol `1`, managed/portable update paths, canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` release assets are preserved.
