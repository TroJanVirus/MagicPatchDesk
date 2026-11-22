# Magic Patch Desk v0.14.0-test.12 — JMP to IPS without an original

## Convert Patch improvements

Selecting a JMP from **Patch Library**, importing one, or dropping one into **Convert Patch** now defaults to **IPS**, not All Formats Pack. If you explicitly choose another format, your choice stays selected.

- **Offset-only JMP → IPS:** no original XBE required. Absolute offsets and bytes are encoded directly, preserving enabled record order and RLE choice. Disabled records are not applied. Encrypted IPS works similarly with a valid password.
- **JMP with search-and-replace records → IPS:** conversion without the original is blocked. The converter explains that the original XBE is needed to locate matching bytes; with the correct original supplied, conversion can proceed and verify the result.
- **JMP with APPEND records → IPS:** an original is needed to establish the end-of-file offset. IPS format restrictions (including its 24-bit range and reserved EOF marker offset) remain enforced.
- **JMP → BPS:** standard BPS needs original-file length and CRC32, plus the target CRC32, **even if the JMP contains only absolute offsets**. A valid BPS cannot be constructed from a source-free JMP alone. The converter now states this explicitly. If the JMP also contains search-and-replace, that reason is named too.
- **JMP → All Formats Pack:** still needs original data for BPS/Xdelta. Per-target messages now identify search-and-replace, APPEND, or the source-dependent output format.

When conversion is blocked, the footer shows the reason instead of a generic "source required" message; clicking the blocked control opens the complete explanation. This does not bypass validation or write an incomplete patch.

## Verification

Added portable regression tests for JMPs imported or chosen from the library, source-free IPS/EIPS output and exact target-byte results, search-and-replace refusal without an original, source-bound BPS generation, and multi-file patch-plan explanations. The Windows UI smoke test also checks default IPS selection, preserved explicit BPS choice and visible search-and-replace blocking.

Original XBE and any supplied modified XBE are never overwritten. The existing independent byte-for-byte conversion tests, updater protocol v1, canonical Windows release assets and previous rollback versions remain unchanged.

The owner's human-written README was deliberately **left unchanged**. Native Windows smoke tests are compiled but are not executed in Linux CI; no claim is made about physical Windows GUI or Xbox hardware testing. The unrelated crash investigation remains open.
