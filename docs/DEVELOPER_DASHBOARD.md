# Developer diagnostics workspace — v0.10

The **Diagnostics** tab is a read-only, triage-oriented view of reports submitted to
`TroJanVirus/MagicPatchDesk` GitHub Issues, alongside the current computer's
existing private reports. It is intentionally separate from **Settings → Problem
Report Groups**, where individual users inspect and choose whether to share
local reports. No report is transmitted simply by opening the Diagnostics tab.

## Data access and privacy

- GitHub data is fetched **only when Refresh GitHub is clicked**. The app makes
  anonymous, bounded GET requests to GitHub's public Issues REST API; no
  access token, GitHub write permission, webhook, telemetry daemon, or
  GitHub Actions is required.
- Results are kept **in memory**, not written to a new user cache. Local JSON
  reports remain under the existing `Reports` folder and are never uploaded.
- Only public GitHub Issues explicitly containing recognised group IDs are
  indexed. The maximum fetch is 5 pages × 100 records = 500 issues **and PRs**.
  A visible partial-results warning appears if the cap is hit. Rate limits,
  offline errors and network failures leave local reports usable.
- The dashboard counts GitHub **report issues**, not unique people, GitHub
  comments or confirmed failed attempts. "Local occurrences" counts only the
  reports on this particular computer. These are never combined into a claimed
  number of distinct affected users.
- Any GitHub text may have been authored by third parties. Links are always
  generated using this repository's fixed HTTPS URLs and validated issue/PR
  numbers; arbitrary remote links are never followed automatically.

## Matching and tracking

A user-report issue can be recognised by the title:

```text
[Diagnostic MPD-0123456789ABCDEF] checksum_mismatch / patch.apply
```

Or by a structured body line:

```text
Diagnostic group: MPD-0123456789ABCDEF
```

Each group key is based on the **category + operation + Title ID** already
established in v0.9. Group IDs are matching hints, not proof of a shared cause.

An optional developer-maintained issue can track the underlying bug across
multiple group IDs. Write the following *in the root issue body*:

```text
[Root cause] Fix source selection in multi-file conversion

Related groups: MPD-0123456789ABCDEF, MPD-1234567890ABCDEF
Fix PR: #42
```

To attach an individual report to that root issue, add the line below **to
that report issue's body**:

```text
Root cause: #123
```

`#123` is the root-cause issue number in the same repository. An individual
report and a root issue with the same group ID are displayed under one
case; cases with multiple group IDs are combined when they explicitly name
the same root issue. **Contradictory root issue references are flagged** instead
of silently merged.

The dashboard shows a referenced root issue's GitHub state (`open`/`closed`)
and a referenced fix PR's GitHub state (`open`/`closed`), if present in the
fetched pages. A *closed PR does not mean merged* and a *closed issue does not
necessarily mean fixed*. Links absent from the fetched pages are explicitly
marked unverified, not assumed closed or merged.

## Native Windows workflow

1. Open **Diagnostics** and click **Refresh GitHub**. Local reports are
   visible even without an internet connection.
2. Sort by **Most reports**, **Newest**, or **Open first**.
3. Select a case to see its group IDs, public issue count, local count,
   root issue and fix PR.
4. **Open report**, **Open root issue**, and **Open fix PR** open your browser
   for viewing or manual editing. **Copy case** copies a safe summary.
5. **Copy group markers** copies a `Related groups: ...` line. Paste this in
   the root-cause issue body; then use **Copy root marker** to paste
   `Root cause: #123` into related individual diagnostic issues.
6. **New root issue** opens a *prefilled but unsent* GitHub issue. Nothing is
   created until you review and submit the form in your browser.

This is not a privileged GitHub issue management client: no issue is automatically
created, edited, closed, labelled, merged or commented on. Maintainers should
verify proposed links before adding them to the issue bodies.

## Verification and limitations

`core/developer_dashboard_test.go` contains fixtures for paginated GET requests,
non-POST behavior, grouping, cross-group root causes, mismatched roots,
fix PR linking, sorting, PR exclusion, malformed IDs and error responses.

Run `go test ./...`, `go test -race ./core`, `go vet ./...`, and
cross-compile for Windows x64. These do not replace real Windows mouse,
clipboard, network, browser, and Xbox/xemu checks.

The unauthenticated GitHub API is rate-limited; manual browser review remains
available if refresh fails. Grouping does not currently inspect comments or
GitHub Projects fields: maintain link markers in issue *bodies*. On a
later version a separately authorized GitHub connector could enable explicit
maintainer actions; no automatic actions are implemented in v0.10.
