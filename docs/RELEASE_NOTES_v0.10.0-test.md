# Magic Patch Desk v0.10.0-test.1 — Developer diagnostics dashboard

Portable native Windows x64 prerelease. **Original Xbox game data is never
included or transmitted**.

## New: Diagnostics workspace

- New native **Diagnostics** tab with current-PC diagnostic groups plus an
  on-demand public GitHub issue snapshot for `TroJanVirus/MagicPatchDesk`.
- Sort cases by GitHub report issue count, newest activity, or open report count.
  Clearly separate local occurrence counts from public GitHub issue counts.
- Group several diagnostic categories under one underlying root-cause issue
  using `Related groups: MPD-..., MPD-...` and `Root cause: #123` markers.
- Show open/closed state for the root issue and a linked `Fix PR: #42` when the
  corresponding records are in the public issue snapshot. No false claim that
  closing a PR means it was merged or that closing an issue means it was fixed.
- Flag conflicting root-cause references rather than merging questionable
  reports. Provide browser links, copyable safe triage summaries, and prefilled
  root issue templates for manual review and submission.
- Fully offline local grouping is preserved. No automatic uploads, background
  polling, GitHub write tokens, privileged API calls, or recurring Actions.
- Fetch at most 500 public issue/PR records and indicate a partial result.
  GitHub API rate limits and network failures do not affect patch application.

## Reliability

- Go core unit tests cover grouping, conflicts, sorting, paginated GET behavior,
  rate limits, privacy of local reports and source-less operation.
- The v0.9 report generator, patch engine, PF support, all-format patch packs,
  previews, source checks and UI hover effects remain unchanged.

## Verification

- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- Cross-build Go 1.23 native Windows x64 portable EXE.

Manual Windows UI, network access, issue-linking workflow and actual Xbox/xemu
validation remain to be tested. Read `docs/DEVELOPER_DASHBOARD.md` for the
marker convention and scope.
