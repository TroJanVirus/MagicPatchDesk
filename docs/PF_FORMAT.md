# Rocky5 / XBMC4Gamers `.pf` compatibility

Source material (Rocky5):
- GUI catalogue: https://github.com/Rocky5/XBMC4Gamers/blob/master/Mod%20Files/system/scripts/XBMC4Gamers%20Extras/File%20Patcher/default.py
- **Actual byte/file engine:** https://github.com/Rocky5/XBMC4Gamers/blob/master/Mod%20Files/system/scripts/XBMC4Gamers%20Extras/File%20Patcher/resources/lib/default.py
- Format reference: https://github.com/Rocky5/XBMC4Gamers/blob/master/Mod%20Files/system/scripts/XBMC4Gamers%20Extras/File%20Patcher/readme.txt
- Example patches: `resources/lib/patches/` in that repository.

## Parsed syntax

Headers appear on `#` comment lines. `Author`, `Credits`, `Info`, `Region`, `Title`, `TitleID` (or `Title ID`) and `Type` are stored. Remaining comments are retained as per-record descriptors. Header case is insensitive; whitespace after the separator is optional.

| Opcode | Meaning and syntax | Implementation |
|---|---|---|
| `CP` | `CP\|oldfile\|newfile` | Copy existing original to a new target; never overwrite existing target |
| `HR` | `HR\|file\|count\|findHEX\|replaceHEX[\|count\|findHEX\|replaceHEX...]` | Replace up to `count` occurrences in file order; multiple triplets supported; missing patterns reported as warnings |
| `OF` | `OF\|file\|decimalOffset\|bytesHEX[\|decimalOffset\|bytesHEX...]` | Exact decimal offset, extending with zeros as needed |
| `SM` | `SM\|file.xbe\|C00001FF` | XBE certificate supported-media mask, little endian in certificate at +0x9C |
| `SR` | `SR\|file.xbe\|80000007` | XBE certificate supported-region mask at +0xA0 |
| `MV` | `MV\|old\|new` | Move/rename file |
| `RN` | `RN\|old\|new` | Rename file |
| `RM` | `RM\|file[\|file...]` | Delete named files |

When used on an original Xbox, Rocky5's Python script uses XBMC UI dialogs and modifies game files in place. Magic Patch Desk does **not** execute Python and does **not** modify original files. It loads the PF as data, validates paths, validates Title ID where appropriate, simulates operations in memory and writes only to independently staged output. Source directory application makes a full game-folder copy before applying changes; users must have enough free storage. If full copy fails, temporary output is removed and source unchanged. All symlinks are rejected.

**Important:** The Rocky5 engine collects `CP` records separately and executes them before other operation types. Its Python 2 `dict.iteritems()` dispatch gives the remaining types an unspecified ordering. Magic Patch Desk explicitly applies `CP` first and then the other records in source order, which is deterministic and better suited to auditing, but not a claim of byte-for-byte interpreter control-flow emulation. The original buffered `HR` path differs from its in-memory path; Magic Patch Desk implements the full-file intended semantics and supports variable-length replacement.

### ISO limitation

Direct XISO application supports only `HR`, `OF`, `SM`, `SR` on uniquely identifiable **existing** XDVDFS file entries. PF scripts that copy, rename, delete or move game files require an extracted game folder. We will not silently ignore these commands or claim to rebuild ISO directory trees.

### Conversion policy

- PF -> JMP is allowed when PF has only byte edits (`HR`, `OF`, `SM`, `SR`), with originals required when effects cannot be translated faithfully without context (HR >1, changing byte lengths, SM/SR certificate offsets). Multiple files are kept in a **single JMP** using per-record filename comments.
- JMP -> PF supports first-occurrence fixed-width hex search/replace and offset writes. JMP APPEND has no native PF equivalent: export is refused.
- PF containing any `CP`, `MV`, `RN` or `RM` is **not** converted to a plain JMP or binary pack; these filesystem operations would be lost. Such patches should be applied natively from PF to an extracted game folder.
- A PF containing only byte edits can also be converted into a `magicpatchdesk-pack-v2` all-formats archive when all original files are supplied.

Untrusted PF input can contain arbitrary file names; these names are always normalized to safe relative paths. No `exec`, shell command, Python execution or absolute path is supported. Every output is separately staged to avoid corrupting originals.
