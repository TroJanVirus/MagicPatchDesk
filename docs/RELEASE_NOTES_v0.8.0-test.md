# Magic Patch Desk v0.8.0-test.1 — Diagnostic reporting

This is a **developer test** of the native Windows app.

## New

- Consistent, local-only diagnostic reports for unsuccessful JMP, PF, IPS/BPS/Xdelta conversions, patch packs, and targeted patch operations.
- A structured `magicpatchdesk-diagnostic-v1` JSON schema with a randomized report ID, error classification, operation, app/OS version, format and CRC metadata (when known), and numeric record/offset details.
- Sensitive material deliberately excluded: no game binary contents, patch bytes, passwords, username, absolute path or unfiltered error message in the report.
- PF operation warnings can produce warning-severity reports even when the overall patch completes.
- Settings > Problem Reports offers **View last report**, **Copy summary**, **Open reports folder**, and **Report to developer**.
- Report to developer opens a *prefilled* GitHub issue after a warning; the user must inspect and explicitly submit it. **No automatic telemetry or background upload.**
- Reports stay in the app's existing local data directory under `Reports`.

## Limitations

- Not every UI/operating-system error is classified beyond `operation_failed`. Unknown raw errors are not included in shareable reports to avoid inadvertently disclosing private data.
- The Win32 clipboard, browser and click interactions were compiled but not smoke-tested on an actual Windows desktop.
- No claims about Original Xbox boot/game effectiveness.

## QA

- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- Cross-compile Windows x64 executable with Go 1.23+
