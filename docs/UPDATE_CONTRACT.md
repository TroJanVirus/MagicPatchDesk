# Magic Patch Desk — permanent self-update contract (protocol v1)

**Never break existing update clients.** This contract was introduced in v0.12.0-test.1. A pre-v0.12 portable EXE does not have the updater; users install v0.12 once and can update from the new Maintenance tab thereafter.

## Program and data directories

Installed program: `%LOCALAPPDATA%\Programs\MagicPatchDesk\MagicPatchDesk.exe`.
The app also copies a version-agnostic `MagicPatchDeskUpdater.exe` helper into the same folder. When a self-update succeeds, the former application is retained as `MagicPatchDesk.previous.exe`. The helper is intentionally a copy of the app containing the stable `--mpd-apply-update-v1` entry point, invoked only after the parent process exits. It validates staged SHA-256 and the managed destination, swaps the main executable, keeps the backup, and restarts. The app's `--mpd-healthcheck-v1` must always exit successfully without opening a GUI.

On first launch, use **Maintenance → Install app** to copy the existing loose EXE into the above folder. The portable EXE remains usable if installation is declined. If the app already has an installed copy, the loose copy must not overwrite it.

Data is **never** stored inside the managed program directory:

```
%APPDATA%\MagicPatchDesk\
  Custom\                 user-imported JMP/PF and artwork
  GitHub\                 synced catalogue cache
  Covers\                 downloaded art cache
  Reports\                local privacy-filtered diagnostic JSON and submission state
  settings.json            existing worker/filter preferences
  recent-files.json        local private path history
  maintenance.json         update channel / auto-check / opt-in report setting (NO TOKENS)
```

The GitHub Issues token is stored separately in **Windows Credential Manager**, key `MagicPatchDesk/GitHubIssues-v1`, never in app data, release assets or diagnostics.

## Mandatory requirements for EVERY future GitHub Release

1. Increment `core.AppVersion` to the release tag (semantic version incl. preview suffix). Do not change `core.UpdateProtocol` unless a backward-compatible protocol is simultaneously maintained.
2. Attach one native Windows x64 executable under **exactly** `MagicPatchDesk-Windows-x64.exe`. Optionally attach a versioned alias, for older links. Neither asset may be a ZIP.
3. Attach `SHA256SUMS.txt` containing the SHA-256 for the **canonical** Windows executable. The current client additionally checks GitHub's asset digest when supplied.
4. Never remove or change `--mpd-apply-update-v1` and `--mpd-healthcheck-v1`. Old updater helpers launched by older EXEs must remain able to launch future builds.
5. Keep GitHub repository `TroJanVirus/MagicPatchDesk` (or provide a tested migration redirect). Updater selects the highest numeric release version, including previews when enabled; GitHub `/releases/latest` excludes previews and MUST NOT be the sole discovery method.
6. Keep the managed EXE name, app-data location and Settings migration/backward compatibility. A valid update must not erase `%APPDATA%\MagicPatchDesk` or Windows Credential Manager data.
7. Test `go test ./...`, `go test -race ./core`, `go vet ./...`, cross-compile Windows GUI, and use `scripts/verify_release_contract.sh` in the one-shot release workflow.
8. Retain `MagicPatchDesk.previous.exe` so users can manually launch the previous version after a regression. Do not force automatic silent updates.
9. If no valid asset/checksum exists, update must **fail closed** and keep the old version, never downloading or running an unverified binary.

No self-updater can mathematically guarantee future releases will never contain errors. The above compatibility contract, release-stage tests, checksum verification and rollback executable are intended to reduce that risk.

## Release discovery

Fetch `https://api.github.com/repos/TroJanVirus/MagicPatchDesk/releases?per_page=100`, ignore drafts, parse numeric vMAJOR.MINOR.PATCH[-preview.N] and choose the highest valid release for the selected channel. Preview releases are included by default until a stable version exists. Stable-only mode excludes previews.

## Limitations

The current executable is not Authenticode signed; the published checksum and GitHub asset digest detect tampering but do not replace publisher code signing. Updating requires HTTPS network access and sufficient space for the staged EXE plus rollback.
