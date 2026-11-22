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
| **GitHub REST API / GitHub Docs** | Existing GitHub Issues transport extended for explicit, consented compatibility submissions; pinned public GitHub raw-content endpoint for a reviewed JSON report index. **Protocol and documentation reference only; no GitHub sample implementation source copied.** | [GitHub Issues API documentation](https://docs.github.com/en/rest/issues/issues); [GitHub API terms](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service) |
| **ConsoleMods Wiki community — Original Xbox motherboard information** | Well-established Xbox motherboard version labels used for structured physical-console field reports (1.0–1.4, 1.6 and 1.6b) and the distinction between a known revision and a user-selected unknown. **Factual reference only; no wiki text, images or software copied.** | [Xbox motherboard information](https://consolemods.org/wiki/Xbox:Motherboard_Information), [Versions](https://consolemods.org/wiki/Xbox:Versions); [ConsoleMods Creative Commons Attribution / site disclaimer](https://consolemods.org/wiki/ConsoleMods_Wiki:General_disclaimer) |
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

## v0.14.0-test.3 PatchHub screenshots

**JayYardley / Xbox Patch Hub** — official source of the game cover and, when publicly exposed, original patch comparison screenshot URLs. [PatchHub](https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php). **Metadata/artwork provenance only.** The app uses its own Go parser and native GDI+ viewer; no PatchHub source, copyrighted screenshots or third-party image assets are bundled in the app or copied into this source repository. User-imported local `titles/` folders retain their original provenance. Do not imply images from an unrelated game version certify this patch.

## v0.14.0-test.4 Windows crash diagnostics reference

**Microsoft — Windows Error Reporting user-mode dumps** ([Collecting User-Mode Dumps](https://learn.microsoft.com/en-us/windows/win32/wer/collecting-user-mode-dumps)) — documentation reference on Windows native dump collection, HKLM administrative configuration and diagnostic privacy. No Microsoft crash library, code or copyrighted data was copied. The app uses its own Go/session-marker mechanism and does not change HKLM settings without administrator action.

## v0.14.0-test.6 — owner-supplied JMP logo and website comparison media

**JayYardley / original Jay's Magic Patch branding** — the transparent JMP logo was explicitly uploaded for embedding in the application's Windows EXE. Its text-encoded source lives in `assets/jmp-logo.ico.b64`; resource conversion is implemented in-house using `scripts/build_windows_icon.py` (Python standard library). The embedded logo is the only newly bundled artwork; no third-party font or screenshot is embedded. PatchHub comparison images remain externally hosted on the author's PatchHub site, downloaded and validated only as needed and cached locally on the user's device. The optional media-index generator reads public website images, not copyrighted game content.

## v0.14.0-test.17 crash investigation (2026-11-22)

**The Go authors / Go team — `encoding/json` documentation and source** ([decoder](https://go.dev/src/encoding/json/decode.go), [package documentation](https://pkg.go.dev/encoding/json)) — referenced to interpret a Go runtime decoder invariant panic found in an uploaded Magic Patch Desk crash trace. **Reference only**; no Go standard-library implementation code copied. Go source: [BSD-style licence](https://github.com/golang/go/blob/master/LICENSE). The new recovery boundary and private crash breadcrumbs are independently authored Go code.

## 2026-11-22 Win32 pointer lifetime reference

**Go team — standard Go Windows syscall and unsafe.Pointer rules** ([`syscall.LazyProc.Call` with `//go:uintptrescapes`](https://go.dev/src/syscall/dll_windows.go), [official `unsafe.Pointer` documentation](https://pkg.go.dev/unsafe#Pointer)) — reference-only evidence for preserving the compiler's pointer lifetime semantics through the application's custom Win32 `sysCall` helper. The helper now explicitly has the same compiler pragma. No external implementation code was copied; Go language/runtime project [BSD-style licence](https://github.com/golang/go/blob/master/LICENSE).
