# Magic Patch Desk .mpdpack — all-formats version 2

`magicpatchdesk-pack-v2` is a ZIP archive of independently addressable binary patches. It **never embeds original Xbox game contents**. It is backward-compatible at the reader level with existing `magicpatchdesk-pack-v1` archives.

A full-all-formats export from one JMP includes:

```
manifest.json
source/original.jmp
patches/default.xbe.ips
patches/default.xbe.bps
patches/default.xbe.xdelta
patches/default.xbe.eips      (only when a password is supplied)
patches/config/settings.cfg.ips
patches/config/settings.cfg.bps
patches/config/settings.cfg.xdelta
patches/config/settings.cfg.eips  (password required)
```

The v2 manifest has one `entries` record per **(target, format)** pair, with patch SHA-256 and the original and resulting file SHA-256. `original_jmp` and `original_jmp_sha256` preserve the complete original JMP. The new `omitted_formats` list records honest omissions, such as IPS size restrictions or missing optional EIPS password. The builder requires the original of **every enabled target** so it can generate BPS and Xdelta, calculate source/target fingerprints, and verify that every generated patch produces exactly the intended bytes.

`ApplyPatchPack` accepts both v1 and v2. For v2, it selects a single verified variant for each target (BPS first, then Xdelta, IPS, encrypted IPS). A failed BPS or source checksum is fatal: the application will not fall back to a weaker patch merely to bypass validation. Clients may extract any contained alternative for use with compatible external patch utilities.

No archive can be called "all formats" if any target lacks a usable binary patch. Non-compatible formats are recorded as omissions; the complete original JMP stays available as the authoritative script.
