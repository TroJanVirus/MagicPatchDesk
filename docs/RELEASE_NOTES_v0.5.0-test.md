# Magic Patch Desk v0.5.0-test.1 — File-aware JMP patching and strict conversion requirements

This is a native Windows developer build. Please test using a **copy** of a game before trusting it with valuable saves or game data.

## Feature: JMP records can target named files

JMP comments can name the intended target, e.g.:

```text
#Widescreen
0x1234:AABBCCDD
#Video mode (settings.cfg)
766964656F3D34383070
766964656F3D37323070
#Other data [file=Media/settings.cfg]
0x10:0000
```

`#Widescreen` applies to the default XBE; the records mentioning `settings.cfg` apply to that file instead. Comments naming `default.xbe` also apply to the default XBE. Group toggles still group identical comment labels (case-insensitive) and `[-]` remains disabled by default.

Before running a multi-file JMP, the app opens an explicit dialog for each extra filename:

- **Yes**: locate the named file on disk, or use a uniquely identified file already found inside the ISO.
- **No**: skip all records targeting that file.
- **Cancel**: cancel the entire patch.

A sidecar loaded from disk is saved in a separate `<chosen_output>_patched_files/` tree, never over the input sidecar. Inside an XISO, the named XDVDFS file can be patched directly in a newly generated ISO. Ambiguous ISO basenames require an unambiguous relative path in the comment; the app never picks at random. Game-partition XISOs can relocate growing files. Full-disc ISOs refuse growth beyond a file's allocated sectors.

The target is extracted conservatively from valid relative filename.ext tokens. Absolute paths, parent traversal, Windows device names and comments with more than one candidate filename are rejected; URLs and common web domains are ignored. File references are never treated as commands to write to arbitrary filesystem paths.

**Limitations:** a JMP cannot automatically invent a file that does not exist inside an ISO. If absent, the user may locate an external file and get a patched sidecar output. Multi-file JMPs are not exportable directly to single-file formats such as IPS: separate records into per-file JMPs first. The XBE's CRC32/Title ID in the JMP metadata is checked only against the default XBE, not a secondary configuration file. An explicit named-file dialog is always required for these records.

## Feature: Strict conversion button

The **Convert / Create Patch** button is visibly disabled when the input format requires source bytes and the original file has not been selected. The explanation shown is:

> This patch conversion requires the source file for the context to create it

A JMP consisting only of offset records can still become IPS/EIPS without the original (verification recommended). JMP search/replace and append require original bytes before conversion to IPS. BPS/Xdelta creation and decoding always require source data in this app. Encrypted IPS inspection avoids expensive password derivation on every keystroke, and certain EIPS conversions are conservatively source-required until their structure can be verified. One JMP containing records for several files cannot become one IPS; the tool rejects that conversion explicitly.

## Safety

- Standard single-file `Apply` and `Preview` now *reject* multi-target JMPs rather than silently applying named-file offsets to default.xbe.
- A new multi-file workflow resolves each target, displays a per-file preview and asks for confirmation. Original files are unchanged.
- Newly generated sidecar output trees must not already exist; no silently overwriting a sidecar.
- File selections are checked against comment filenames. Input names are normalized and checked for traversal.
- Conversion protections remain in place; BPS and Xdelta compatibility limits are unchanged.

## Automated validation

`go test ./...`, `go test -race ./core`, `go vet ./...`, Windows x64 cross-compilation. Includes synthetic XISO multi-file patch tests and full-disc relocation refusal, filename detection, unsafe path rejection, and source-conversion requirements.

**Unverified:** GUI dialogs on physical Windows, patched Xbox game boot behaviour, and full-disc ISO compatibility beyond the test fixtures.
