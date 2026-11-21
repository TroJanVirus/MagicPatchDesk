# Optional privacy-redacted reports to GitHub Issues

The application always saves diagnostic reports locally in `%APPDATA%\MagicPatchDesk\Reports`. **Automatic sharing is OFF by default.** To enable:

1. Open **Maintenance** and choose **Connect token**. Generate a GitHub Personal Access Token with enough permissions to create issues and comments on `TroJanVirus/MagicPatchDesk`, copy it to the clipboard, and confirm. The app saves it in the current Windows user's Credential Manager, not in app settings, then clears the clipboard text.
2. Choose **Auto-report**. The app now attempts to upload new privacy-filtered reports and up to five queued reports on subsequent launch. Failed submissions remain local for retry using **Sync now**.
3. **Disconnect** removes the stored credential and disables further automatic submissions. Existing submitted GitHub issues remain public; manually remove or edit those on GitHub if needed.

A report's stable `MPD-` group ID is used to search GitHub Issues. The first report creates a `[Diagnostic MPD-...]` issue; later reports create comments on a matching issue. Report IDs are remembered in a local `Reports/submission-state.json` to avoid re-submitting the same event. There is no GitHub Actions workflow attached to incoming issues.

**Only the redacted `core.DiagnosticReport.Summary()` and safe metadata are submitted.** Never include game binaries, source patches, user paths, executable bytes, passwords, or raw exception strings. Reports are public once posted. Review the first report on GitHub to confirm it contains what you expect.

If you have connected your GitHub account to ChatGPT, ask: **“Check the latest diagnostic reports in TroJanVirus/MagicPatchDesk issues; group the problems, prioritize fixes and identify duplicate root causes.”** ChatGPT can read the public issue corpus directly. The app does not grant ChatGPT any special access or turn on monitoring by itself.

GitHub credentials are required because ordinary anonymous clients cannot create Issues through the API. Failure to connect a token does not interfere with any patching or local report logging.
