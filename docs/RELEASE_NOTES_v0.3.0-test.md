# Magic Patch Desk v0.3.0-test.1 — Patch conversion and BPS

## Changes
- A dedicated **Choose patch** control in Convert & Create accepts JMP, IPS, Xdelta/VCDIFF, encrypted IPS (EIPS), and BPS.
- Choose an original stock XBE plus a source patch, or instead choose a modified XBE. The patch is applied to the stock bytes in memory and the result is encoded in the selected output format.
- Output formats now include **BPS** alongside JMP, IPS, Xdelta and EIPS.
- BPS encoder produces valid BPS1 with SourceRead/TargetRead; decoder reads all four BPS actions (SourceRead, TargetRead, SourceCopy, TargetCopy), including overlapping copies. Source, target and patch CRC32 are validated.
- **Standard IPS RLE encoding** is selectable on/off for IPS and encrypted IPS (EIPS). RLE is standard IPS, not encryption; EIPS remains the custom AES-GCM password-encrypted option.
- Fixed IPS reserved 'EOF' offset case for output generated at 0x454F46.
- Source patch and modified-XBE choices are mutually exclusive; custom source patches can be chosen directly without first importing into the library, or dropped on the Convert page.
- Protects original XBE, modified XBE and selected source patch from being overwritten by conversion.
- Improved conversion error messages for invalid format or mismatched output extension.

## How to test
1. Open **Convert & Create**.
2. Select the unmodified original XBE used to make the patch (BPS requires exact source bytes and CRC).
3. Click **Choose patch** and select an existing .xdelta, .jmp, .ips, .eips or .bps file. Alternatively select a modified XBE instead.
4. Choose **IPS** (or JMP, BPS, Xdelta, EIPS) as output; toggle **IPS RLE encoding** if desired.
5. Supply a password only for encrypted IPS and click **Convert / Create Patch**.
6. Verify the converted patch applied to the same stock XBE produces identical output bytes.

## Notes and limitations
- **All conversions require the original XBE.** Arbitrary binary delta patches cannot be reliably converted without reconstructing their target against the correct source.
- Existing Xdelta decoder handles only the documented uncompressed VCDIFF/default-code-table subset, not all external xdelta3 files.
- Standard IPS has a 24-bit maximum output size (roughly 16 MiB). For larger images use BPS or supported Xdelta; JMP can append but cannot truncate.
- Encrypted IPS is the custom EIPS format used by Magic Patch Desk, not an interoperable ISO-standard encrypted IPS.
- No actual Windows GUI smoke test or real Xbox game boot test has been performed yet.

## Validation
- go test ./... (includes 25 source-patch/output-format combinations on synthetic data)
- go test -race ./core
- go vet ./...
- Windows x64 PE GUI cross-compilation
