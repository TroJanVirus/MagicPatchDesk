# Magic Patch Desk v0.14.0-test.13 — Home, split discs and drag-and-drop game matching

## A proper landing page

Magic Patch Desk now starts on **Home**. It briefly explains patch matching, the available game-output formats and that you can **drag and drop a game or patch anywhere in the app**. Use **Open Xbox game** or **Browse patches** to begin. **Visit Patch Hub** opens the official website in your browser: https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php.

Home is a native workspace, not a webview. All existing sidebar destinations and updater controls remain available. The owner-written README has **not been changed**.

## Split game images

- **Cerbios CCI:** `Game.1.cci`, `Game.2.cci`, ... were already detected as one set, but the picker only validated the container; it didn't identify `default.xbe`. The app now reads the complete split set as a read-only logical stream, parses its XDVDFS filesystem and reads **only the XBE sectors needed** for game title, Title ID and CRC32. Selecting part 1, 2 or a later part gives the same identity. Named patch targets also read directly from CCI without fully extracting the game just to verify a patch.
- **Numbered raw ISO/XISO:** `Game.1.iso`, `Game.2.iso`, ... and the corresponding `.xiso` naming are now read as one virtual disc. The parser and member extractor work even when a file crosses part boundaries, and you may select any available part.
- Each set requires its first part and contiguous numbering; gaps, missing first parts, invalid CCI indexes and corrupted XDVDFS layouts are errors, not success. The Patch Library source line identifies the selected part and total (such as **CCI part 2 of 4**).
- When several numbered parts are dropped together, they are grouped into **one game selection** instead of performing separate scans.
- A numbered raw ISO is never copied or directly compressed as though one selected part were a complete image. For numbered raw images, Patch Library can extract the complete game and build new **game-partition XISO, CCI, extracted folder or patched XBE** outputs as supported. Direct full-disc preservation of split raw files remains disabled to avoid false/incomplete images.
- A true full game extraction/repack still needs enough temporary disk space. Only game identification and targeted CCI file reads avoid full temporary disc decompression.

## Drag and drop

Dragging a `.xbe`, `.iso`, `.xiso`, or `.cci` onto the app now switches to **Patch Library**, reads the embedded XBE asynchronously and selects the matching library game based on Title ID, title and available CRC32. A compatible patch is **suggested**, never applied automatically. When the title is recognized but no matching JMP exists, the status shows its title/ID and explains that no patch is available.

Dropping `.jmp` or `.pf` imports the patch as before. Dropping a folder containing a root `default.xbe` works from Home as well as Patch Library.

## Verification and scope

Synthetic cross-sector, cross-part and missing-part tests were added for CCI and numbered raw ISO/XISO streams, including `default.xbe`, named game assets, XBE CRC32, extracting a full split game, rebuilding an independent XISO, and refusing unsafe fragment-only copy or CCI compression. Native Windows smoke checks cover Home navigation and visible entry points.

Recognized naming schemes are **`Base.1.cci`**, **`Base.1.iso`** and **`Base.1.xiso`** with sequential numbers up to the existing 64-part safety limit; arbitrary archive formats or other split filename conventions are **not** claimed supported. If an unknown last part is entirely absent and the remaining image is still internally consistent, a filename-only directory scan cannot prove it was originally part of a longer set. Checks still fail if referenced files extend beyond the visible image.

Existing originals are never overwritten. Self-update protocol remains **v1** and canonical asset names stay `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`. No actual Windows UI interaction, real retail image compatibility, or physical Xbox/Cerbios bootability is claimed without real-world tests. The prior unexpected-exit investigation remains open.
