# Magic Patch Desk v0.11.0 test 1 — Recent Files and quick workflow controls

**Developer test build:** Windows 10/11 x64. Original Xbox patch engines, format support, validation and output-copy safeguards are unchanged.

### New in this version

- **Recent Files workspace:** up to 18 private, local history entries for game sources, standalone patches, conversion originals/patches/modified XBEs, and generated output copies. Previous/Next navigation; select and reopen, open containing folder, copy path, forget entry or clear history.
- **Safer handling of outputs:** opening a generated output via Recent takes you to its folder instead of quietly feeding the modified file back into the patch engine. New outputs from JMP, PF, pack and conversion workflows are recorded only after successful writes.
- **Quick keyboard controls:** Ctrl+O (game), Ctrl+P (patch), Ctrl+F (search), Ctrl+R (recent), Ctrl+1–6 (tabs), F5 (local rescan), Esc (exit text focus).
- **Small-window improvement:** hide the sidebar's sync/status card before it could overlap the final navigation items.
- **No background telemetry:** full local recent paths are stored only in the app's private `recent-files.json`. This is distinct from privacy-filtered diagnostic reports and is not uploaded.

### Verification

- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- Native Windows x64 cross-build
- GitHub one-shot release validation pending before publication

**Manual testing still needed:** mouse/keyboard interactions on real Windows 10/11, file dialog layout/DPI and opening file folders via Windows Explorer. No Xbox real-hardware test has been performed for this UI-only release.
