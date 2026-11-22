# Magic Patch Desk v0.14.0-test.17 — Rapid game-source switching stability

A user reproduced an abrupt Windows exit by selecting a Phantom Crash JMP and repeatedly choosing **ISO → CCI → XBE** game sources in Patch Library. The crash is not limited to either image format. A real Windows crash dump was not supplied, so the exact native faulting instruction remains unknown.

## Underlying overload mechanism identified

Previous releases moved the final game information update onto the Win32 UI thread and checked a generation number before accepting the result. But every click on **Open file** still launched a **new inspection goroutine**. A stale result was ignored **only after all work had finished**. Rapid switching therefore accumulated concurrent ISO XDVDFS scans, CCI index allocations and LZ4 sector decoding, full-XBE memory allocations and open file handles.

### What changed

- **One source-inspection worker per process**, not one per file selection. There is exactly one active inspection and at most one queued replacement; a newer selection replaces any pending older selection.
- **Cancellation of superseded reads** at all expensive points: split CCI index validation, XDVDFS directory traversal, full-XBE reads, and CCI decompression across 2048-byte sectors. An obsolete result cannot publish game identity, output formats or patch-selection changes.
- **Reduced CCI index memory**: the CCIM index is decoded from 32 KiB chunks directly into a single retained `uint32` index, instead of first loading the whole index into a second multi-megabyte byte slice. This reduces per-part allocation pressure without changing the CCI format or disabling multipart support.
- **Clean shutdown** cancels any running inspection and stops the inspection queue when the native window closes. The queue posts only immutable results to the Win32 owner thread, retaining the generation/path guard and previously implemented safe output-format validation.
- **Selected patch and record toggles are preserved** when the image identifies the already selected game. No patch is automatically applied, and original ISO/CCI/XBE/JMP source bytes remain read-only.

### Regression testing

Portable core tests perform **120 complete mixed-format inspections** (ISO/CCI/XBE), checking identical XBE Title ID, whole-file CRC32, game-source type and byte-for-byte unchanged sources. Separate tests verify pre-canceled scans in all three formats and cancellation of XDVDFS and CCI readers.

The actual **Windows GUI smoke test now creates a valid synthetic XBE, XISO and CCI**, selects a matching synthetic widescreen patch, switches between the real on-disk sources **360 times**, and verifies that the final requested CCI then XBE is identified, record/group states are unmodified, **at most one inspection runs concurrently**, and file-handle/memory usage does not grow abnormally. This test uses an actual Windows executable and OS file IO; it does not require any commercial game asset.

### Limits

Synthetic stress fixtures and native Windows smoke tests demonstrate structural stability, but **do not prove the owner's exact Phantom Crash files can never cause a crash**. The user should retest the repeated ISO → CCI → XBE selection sequence on Windows after installing this version. A private Windows Error Reporting dump or runtime trace is still valuable if there is another abnormal exit. The historical root cause is therefore recorded as a plausible, observed workload-overlap bug, not a confirmed faulting instruction.

The user-supplied human `README.md` remains unchanged. Updater protocol **v1**, portable and installed update paths, the canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` assets, patch matching and original-game protection remain compatible.
