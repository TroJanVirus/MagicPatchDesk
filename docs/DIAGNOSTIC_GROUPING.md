# Diagnostic grouping (v0.9)

Local report events each have a random 18-hex-character event ID, but related failures now receive the **same deterministic group ID** (`MPD-` followed by 16 hexadecimal characters). The group is computed from the versioned canonical combination of **classified error code + operation + validated eight-digit Title ID**, not the report's date, file name or raw exception text.

The local Settings > Problem Report Groups page shows groups ordered by recency and includes occurrence counts, warnings, Title ID, and operation. The Previous and Next buttons navigate the groups. All individual JSON reports stay on disk under `Reports/`. The app reads up to 1,000 most recent local reports for grouping; old v0.8 reports without a group ID are grouped by recalculating the same key. Source binaries, patches, absolute paths, passwords and raw error messages are never used in the group key.

## GitHub issue triage

- **Find matches** opens a GitHub issue-search page with the group's stable ID. The user's browser performs the search while logged into GitHub; Magic Patch Desk does not call the GitHub API in the background.
- **Report issue** offers to search for matching issues first (recommended), compose a new issue, or cancel. The search path attempts to copy the safe summary to the clipboard so the user can paste it as an additional comment to an existing issue.
- New issue titles contain the same group ID and a human-readable error category / operation / Title ID. This makes incoming issues discoverable together via GitHub search. A report isn't published until the user explicitly submits it in the browser.
- This version does **not** auto-close duplicate issues, apply privileged labels, post comments or run Actions on issue events. It requires neither secrets nor GitHub Actions quota. The maintainer can assign labels or close duplicates after review.

A missing/invalid Title ID is represented as `UNKNOWN`. Such reports cluster on error category and operation; users can add a correct Title ID to their manually reviewed issue if known. A group is a matching heuristic, not proof that two failures share the same root cause.

## Privacy and limitations

Only already-allowlisted metadata participates in a grouping key. A group ID is a deterministic fingerprint of structured fields, **not an anonymisation guarantee**; anyone who knows its possible inputs can derive the same fingerprint. In addition, new issue titles explicitly include category, operation and Title ID. These details are visible to GitHub and anyone with access to the issue if the user submits it. No data is sent until a user opens the GitHub search or issue page with explicit confirmation.

Grouped local failures do not automatically produce a centralized dashboard; GitHub search groups by searchable issue title. The UI and clipboard/browser behaviour still require hands-on Windows verification.
