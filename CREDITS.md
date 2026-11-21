# Credits, reference work and attribution

Magic Patch Desk is an independent, native Go/Win32 Original Xbox patching and image-conversion application. This document identifies **specific third-party projects consulted for formats, compatibility expectations or interface/workflow design**, what the project used them for, and the boundary between reference and copied implementation. It is also linked from **About → View full credits + source links** inside the application.

**Important:** Attribution is not a substitute for licence compliance. Do not import or redistribute a project's licensed source, assets, binary or proprietary content merely because it is listed here. Unless a future commit explicitly records a licensed dependency or component, these entries denote **reference only**. No Repackinator source code or binary is incorporated into Magic Patch Desk by the image-output work of 2026-11-22.

| Project / people | Specific acknowledgement and use | Authoritative source / licence |
|---|---|---|
| **Team Resurgent — Repackinator**, especially EqUiNoX (lead programmer), HoRnEyDvL (tester/project manager) and Hazeno (tester) as credited upstream | Original Xbox image-management workflows: extract, rebuild/trim game partition, compress images, preserve FATX splitting limits, verify metadata. **Workflow reference only; no GPL implementation code has been copied, linked or bundled.** | [Team-Resurgent/Repackinator](https://github.com/Team-Resurgent/Repackinator); [GPL-3.0](https://github.com/Team-Resurgent/Repackinator/blob/main/LICENSE.md) |
| **Team Cerbios** and **Team Resurgent — libCCI** | Cerbios Compressed Image (CCI) container design and compatibility, CCIM header, sectors, LZ4 and split sets. Referenced by native pure-Go CCI implementation. | [Team-Resurgent/libCCI FORMAT.md](https://github.com/Team-Resurgent/libCCI/blob/main/FORMAT.md); review upstream licence before reusing code |
| **purelogiccode — CCISharp** | Secondary public reference for CCI sector/index layout and compatibility cross-checking. No library incorporated. | [CCISharp format documentation](https://github.com/purelogiccode/CCISharp/blob/master/docs/format.md); review upstream licence before reusing code |
| **Rocky5 — XBMC4Gamers** | \`.pf\` game patch operation conventions and Xbox dashboard workflow interoperability. The project parses/emulates the operation model independently. | [Rocky5/XBMC4Gamers](https://github.com/Rocky5/XBMC4Gamers); [GPL-3.0](https://github.com/Rocky5/XBMC4Gamers/blob/master/LICENSE) |
| **JayYardley — Xbox Magic Patches by Jay / PatchHub** | JMP patch catalogue, original patch metadata, creator and game attribution. Imported patches retain their original author headers. | [Xbox-Magic-Patches-by-Jay](https://github.com/JayYardley/Xbox-Magic-Patches-by-Jay); [PatchHub](https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php). Individual patch creator credits remain attached to each JMP. |
| **Andrew de Quincey — Xbox DVD Filesystem Details (XDVDFS v0.2, 2002)** | Historical field layout for 2048-byte sectors, volume descriptor, four-byte tree indices, balanced directory trees, 0xFF entry padding and empty directories. **Public documentation reference only; no source code incorporated.** | [Original technical document](https://multimedia.cx/xdvdfs.html); no stated code reuse licence in the document; treated as format reference only. |
| **purelogiccode — XISOStudio / XISO technical documentation** | Cross-check of XDVDFS tree pointers and input-validation conventions. **Format/documentation reference only**, no C# implementation code, binaries or libraries reused. | [XISOStudio technical documentation](https://purelogiccode.github.io/XISOStudio/XDVDFS-Technical-Documentation.html); [XISOStudio GPL-3.0](https://github.com/purelogiccode/XISOStudio/blob/master/LICENSE.txt). |
| **Xbox platform / XDVDFS format documentation and community** | Xbox game-partition structure, XBE metadata and ISO/XDVDFS conventions, independently implemented against a native parser and synthetic test images. | [XboxDevWiki](https://xboxdevwiki.net/), including its XDVDFS/XBE technical material; community information is used as a format reference, not a code dependency. |

## Feature provenance: 2026-11-22 image output work

- The **user requested** that supplying an XISO, ISO, CCI or folder should permit an independently chosen output: patched XBE, extracted folder, game-partition XISO, CCI, or full-disc ISO only when safely possible. User specifically identified Repackinator as a useful tool reference.
- Repackinator's **workflow, format distinctions and documentation** were reviewed; its GPL-3.0 code was **not reused**.
- The new implementation in \`core/game_outputs.go\` writes XDVDFS directory tables, extracts safely, stages outputs and validates round-trips using existing Magic Patch Desk internals. It was implemented as new Go source; supporting CCI uses the repository's existing pure-Go \`core/cci.go\`.
- The new native Windows workspace in \`cmd/magicdesk/game_outputs_windows.go\` and \`paint_windows.go\` explicitly chooses outputs and offers attribution in About.
- These are development changes. Real-world disc bootability, Cerbios device compatibility and Windows 10/11 GUI interactions must not be claimed without tests.

## Mandatory credit policy for all future work

Every substantive external reference used to build a new feature must be added **before release**, stating: (1) project and authors when known; (2) direct canonical source URL; (3) which format, behaviour, data, code or assets were used; (4) the upstream licence with a link; and (5) whether work was reference-only, adapted, copied or bundled. Copies and derivatives require licence review, source/original notices and the correct distribution obligations. Keep the user-visible About attribution updated, not just README comments.

When opening a patch from the catalogue, preserve its JMP \`author\` value and source link. Do not conflate the credits for the Magic Patch Desk application with the credits for individual patch creators. Do not claim Team Resurgent, Team Cerbios, Rocky5 or other upstream contributors endorse this application.

## Image-integrity audit, 2026-11-22

The Xbox DVD filesystem technical descriptions by **Andrew de Quincey** and **purelogiccode** were cross-checked while hardening balanced XDVDFS directory entries, filename safety and sector/entry alignment. These technical **format facts** informed the independently authored Go changes. No GPL-licensed XISOStudio source was copied. CCI split-set safety improvements were reasoned from the existing repository's native implementation and synthetic failure tests, without copying new external source.
