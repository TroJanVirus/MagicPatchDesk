# Magic Patch Desk v0.4.0-test.1 — source-aware conversion and cover reliability

## Changes

- Patch conversion now distinguishes **source-required**, **source-optional** and **source-free** paths by inspecting actual patch records, including disabled JMP entries, IPS RLE records, and optional final-file-size operations.
- **Offset-only JMP → IPS/EIPS** works without a stock XBE. Enabled records retain their order and overlapping write semantics. An original XBE is recommended to verify the JMP's CRC32 and Title ID before converting, because IPS has no built-in CRC32 field.
- **IPS/EIPS → IPS/EIPS** can be re-encoded without an XBE, including optional file resizing and IPS RLE; password is still required to decrypt EIPS.
- **IPS/EIPS → offset-only JMP** works without an XBE when there is no explicit IPS final-size extension/truncation. The tool clearly warns that offset bounds and source-dependent CRC32 cannot be verified.
- **Search-replace or APPEND JMP → IPS** requires the original XBE to locate search matches or determine the old EOF. **BPS and Xdelta/VCDIFF** require an original XBE for every conversion. Producing BPS or Xdelta always requires original bytes.
- The converter now shows a format-specific requirement hint. When an original is omitted, unverified source-free conversions require user confirmation. When supplied, the tool applies both the original patch and converted patch to the original bytes and verifies byte-for-byte equality before saving.
- PatchHub cover downloads now validate actual JPEG, PNG or GIF data and reject HTML errors. Previously cached bad files are ignored and replaced. Failed downloads are retryable after a cooldown instead of getting permanently blocked until restart.
- PatchHub cover links support https://jayxbox.com and https://www.jayxbox.com redirects (while blocking redirects to third-party hosts). Covers are cached with a .jpg filename for GDI+ decoder compatibility; valid legacy .img files remain usable offline.
- Matching from PatchHub to Github game titles now tolerates punctuation and separators such as `Halo: Combat Evolved` vs `Halo - Combat Evolved` and refuses ambiguous normalized names. Existing Title ID matching remains intact.

## Limitations

- IPS has a 24-bit address/length limit and no standardized CRC32 metadata, so a source-free direct conversion cannot verify the original file or guarantee correct patching.
- JMP offset records cannot extend an XBE; for IPS -> JMP without a source, the target must already be long enough for every offset write.
- Source-free conversions cannot faithfully translate IPS final-size extension/truncation to JMP, nor JMP search-replace or append to IPS.
- BPS enforces source, target and patch CRC32. Xdelta uses VCDIFF; it may contain an optional output checksum, but **does not inherently guarantee a source-file checksum**. The existing decoder still only supports the documented uncompressed default-table VCDIFF subset.
- The website may legitimately provide no image for some titles; those continue to receive the app's built-in initial-based placeholder.
- Real Windows GUI testing, downloaded artwork on Windows, and Xbox hardware validation remain outstanding.

## Verification

- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- Cross-compile native Windows x64 GUI with Go 1.23
- New regression tests cover conversion preflights, direct/reencoded IPS RLE, disabled JMP records, source validation, BPS/Xdelta requirements, cache validation, normalized cover title matching and PatchHub source markup.
