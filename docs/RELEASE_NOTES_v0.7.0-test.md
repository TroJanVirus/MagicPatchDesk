# Magic Patch Desk v0.7.0-test.1 — All-format packs and Rocky5 PF integration

## All-format JMP export
- Convert a single JMP to one `.mpdpack` with **IPS, BPS, Xdelta/VCDIFF, and optionally encrypted IPS** variants for each named target. Include the original JMP as one file and a v2 manifest with hashes and honest format omissions.
- All-format export requires the correct original of **every** enabled JMP target, including files loaded directly from a selected game-partition XISO or supported full-disc ISO.
- Existing v1 packs continue to apply; v2 packs choose the verified BPS variant preferentially. Binary patch formats remain independently extractable.

## Rocky5 XBMC4Gamers PF support
- Parse full `.pf` headers and all eight documented operation types: CP, HR (multiple search/replace instances and variable-length substitution), OF (decimal offsets), SM, SR, MV, RN and RM.
- PF files are selectable from local library, drag/drop and direct patch picker; locally imported PF metadata, authors and game titles appear in the catalogue.
- Preview PF operations without writing. For loose game files, apply on an independent **full game-directory copy**; title-ID validation, path traversal rejection and staging protect the original.
- Existing-file-only PF byte edits may be applied to XISO files; CP/MV/RN/RM require an extracted folder and are never silently skipped.
- Convert byte-only PF -> one multi-file JMP; source-dependent conversion requires originals. PF file operations cannot be losslessly represented by JMP and are blocked.
- No imported XBMC Python execution, no ZIP assets or copyrighted Xbox games bundled.

## Verification
- `go test ./...`, `go test -race ./core`, `go vet ./...`, native Windows amd64 PE cross compilation.
- Tests include samples based on Rocky5's public Auto Modellista and generic PF files, filesystem copy/move/remove, decimal OF, multi-instance HR, media and region certificate fields, PF->JMP, XISO, v1/v2 pack compatibility and checksum enforcement.
- Native UI and actual game console tests are still outstanding.
