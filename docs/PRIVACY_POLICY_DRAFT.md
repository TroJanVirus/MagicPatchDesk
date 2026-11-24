# Privacy and network activity — draft for owner review

**This is a review draft, not a final or legally verified privacy notice.** It describes the current documented Magic Patch Desk source as of v0.14.0 and identifies points for owner confirmation. It does **not** govern Jay's independently operated Patch Hub website, which may have a separate policy.

## What the app does locally

- Magic Patch Desk runs as a native Windows application and patches **copies** of user-provided Xbox game images. The original game sources are not intentionally uploaded by the patch workflows.
- Settings, downloaded patch catalogue files, validated cover art, patch-output history and diagnostic reports are stored locally under `%APPDATA%\MagicPatchDesk` (exact subfolders are documented in `docs/UPDATE_CONTRACT.md`).
- Recent Files may include local filenames and paths. Raw crash traces can contain sensitive path and process details. They remain **local** unless the user separately chooses to share them.
- Optional GitHub Issues credentials are stored in Windows Credential Manager, not the settings JSON.

## Network requests

- **Default startup behaviour:** the program checks GitHub for application releases and community patch catalogue changes, and refreshes Patch Hub metadata/artwork periodically. These requests occur automatically unless `Maintenance → Manual check + sync` has been enabled. Users can still trigger checks on demand. This is **not** a claim that all network transfers require an individual button click.
- **Patch artwork:** when enabled/needed, HTTPS images from `www.jayxbox.com` may be fetched and cached locally. The source protects supported formats, download sizes and the allowed host.
- **Diagnostics sharing:** most crash and failure reports stay private locally. If the user connects a GitHub Personal Access Token and explicitly enables Auto-report or submits an issue, a filtered report summary can become a **public GitHub Issue/comment**. The operator and GitHub can then access the submitted public information. Auto-report is off by default and can be disabled independently of normal patching.
- **Developer dashboard:** user-triggered requests may read public GitHub Issue data. There is no permission to upload private game files for this purpose.
- **Ordinary web metadata:** services involved in any network request may see the user's public IP address, access time and standard HTTP request metadata. The behaviour of logs on Patch Hub and GitHub is governed by their respective operators' practices and privacy notices. The project cannot guarantee those external log-retention periods.

## Data controls

- `Maintenance → Manual check + sync` disables automatic startup update/sync requests after the setting is saved; previously completed requests cannot be retroactively undone.
- `Maintenance → Auto-report OFF` disables diagnostic auto-submission. Disconnecting the GitHub credential removes the app's stored token; it does not delete an issue already made public on GitHub.
- A user can remove application settings and cached files from `%APPDATA%\MagicPatchDesk` and remove the managed executable from `%LOCALAPPDATA%\Programs\MagicPatchDesk` once the application and updater are closed. Portable copies and their updater/backup files can be deleted from their own directory.
- Do not recommend deleting an original game file. Do not claim deleting a local report deletes any already-public GitHub Issue.

## Owner confirmation needed before publication / SignPath submission

1. Does Patch Hub log IP addresses, user agents or request paths? If so, what privacy policy and retention terms govern that **separate website**?
2. SignPath requires extra user-facing notice and the ability to disable network transfers if the program sends user data to unselected remote systems. The current app may make its first network request **before** a new user visits Maintenance. Decide whether to add a first-run privacy/connection disclosure with a clear offline option, or to ship with startup sync off until users opt in.
3. Confirm whether any app-specific telemetry or third-party endpoint not listed above exists. Check runtime URLs and HTTP request sources before publishing an unqualified “no telemetry” claim.
4. Confirm how users can remove an installed app, backups, caches, credentials and (if applicable) the independent Patch Hub server's logs.

Related services:
- GitHub privacy: https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement
- SignPath Foundation rules: https://signpath.org/terms
- Patch Hub site: https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php

**Do not publish this draft as definitive or claim that the present startup network behaviour meets SignPath's conditions without review.**
