# Magic Patch Desk v0.9.0-test.1 — Diagnostic grouping & duplicate discovery

- Automatically groups matching **local** error and PF warning reports by stable classification, affected operation and validated game Title ID.
- Each diagnostic keeps its random individual report ID; a new deterministic `MPD-...` group ID links reports across sessions and app restarts.
- Older v0.8 JSON reports are grouped without being rewritten or deleted.
- Settings > Problem Report Groups shows severity counts, occurrences and navigation, plus view/copy/report-folder actions.
- **Find matches** opens GitHub issue search for the group ID. **Report issue** offers existing-issue search before drafting a new issue that includes the group ID in its title. Search is done in the user's browser, with explicit confirmation. No automatic submission, telemetry or incoming-issue Action workflow.
- Reorganised the Settings layout for the diagnostic grouping controls and removed redundant last-report state.
- Reports read the newest 1000 local events; the count is a local troubleshooting index, not a global telemetry total.

### Validation
- `go test ./...`
- `go test -race ./core`
- `go vet ./...`
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H windowsgui -s -w' ...`

Windows UI behaviour and actual GitHub issue submission have not been manually verified. Grouping is a heuristic; shared failure groups do not guarantee a shared root cause. Issue creation must be deliberately completed by the user.
