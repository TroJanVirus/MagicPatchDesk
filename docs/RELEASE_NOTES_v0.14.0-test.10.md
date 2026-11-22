# Magic Patch Desk v0.14.0-test.10

This preview is mainly for the end-user README, the Convert Patch screen and crash prevention.

## What changed

- The repository's main README has been replaced **verbatim** with the owner's more human, end-user-focused `README.md`. It has not been padded out with development notes; deeper technical context remains in `PROJECT_CONTEXT.md` and `docs/`.
- **Convert Patch** now puts **01 Source patch** first, **02 Stock / original XBE** second and the optional modified-XBE reference below them. The footer now explains that the modified XBE is a byte-for-byte reference when supplied with the original.
- When converting an existing source patch, the selected modified XBE is checked against the input patch's output, then the **newly converted output** is independently applied to the original and compared byte-for-byte before saving. The existing conversion core also verifies input/output round trips whenever an original is available.
- When **creating** from original + modified XBEs, the generated JMP / IPS / Xdelta / EIPS / BPS is now applied in memory and compared to the expected modified XBE before any file is written. A verification failure blocks saving.
- For patch packs, an optional modified `default.xbe` is compared with the result of applying the generated pack when original `default.xbe` is provided. Pack targets and their hashes continue to be checked independently.
- If an ISO is chosen as the original source, the converter reads **default.xbe inside the XDVDFS filesystem**, rather than mistakenly treating the raw ISO as XBE bytes.
- The About tab ends with **“References are credited, not bundled. No sources have been copied.”** The reference list remains intact.

## Unexpected-exit investigation and mitigation

The user's `v0.14.0-test.9` evidence contains a previous-session marker labelled `import_patch` and a separate later session marked `ui_callback`. A prior marker can remain labelled `import_patch` after the import dialog finishes; that stale stage does not prove import caused a later unexpected exit. Import, selected-patch parsing and automatic game matching now restore the ordinary UI stage after completion.

**Concrete concurrency risk mitigated:** the app previously swapped its catalog slices, selected patch and parsed patch **from background sync/scan goroutines** while the native Win32 paint path read them without holding the same lock. Catalogue reload and automatic XBE match results are now delivered via `WM_APP` messages and applied on the GUI owner thread. Heavy scans still run off-thread; no patch data or game file is changed.

**Not claimed:** that these changes prove or definitively fix the reported Windows CTD. A private runtime log or Windows Error Reporting dump is still needed to determine the actual fault and instruction pointer. The unexpected-exit marker alone cannot establish this.

## Compatibility and tests

Updater protocol v1, portable/managed updates, existing cached JMP files, `Custom`, game originals, rollback, canonical asset `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` remain unchanged.

Offline tests added for byte-perfect reapplication of all five patch formats, mismatch rejection and the native source-before-original UI hit ordering. The native UI smoke assertions are compiled in the Windows binary but cannot be executed on Linux. Build/test success does **not** establish physical Windows GUI stability or Xbox hardware compatibility.
