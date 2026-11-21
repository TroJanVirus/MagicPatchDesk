package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// DiagnosticGroup summarises recurring failures without storing original
// filenames, binary data, search patterns, credentials, or raw error messages.
// A group is determined ONLY by the failure category, operation and game Title ID.
type DiagnosticGroup struct {
	ID               string
	ErrorCode        string
	Operation        string
	TitleID          string
	Occurrences      int
	Errors           int
	Warnings         int
	FirstOccurredUTC string
	LastOccurredUTC  string
	Latest           DiagnosticReport
}

func canonicalDiagnosticTitleID(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !diagnosticTitleIDRE.MatchString(s) {
		return "UNKNOWN"
	}
	return s
}

// DiagnosticGroupID is a deterministic, non-reversible identifier. The
// category/operation/TitleID are strictly allowlisted before hashing. The
// identifier is stable across sessions and machines; the per-event report ID
// remains random. Version the canonical encoding if grouping rules ever change.
func DiagnosticGroupID(category, operation, titleID string) string {
	source := "mpd-diagnostic-group-v1\x00" + diagnosticTag(category) + "\x00" +
		diagnosticTag(operation) + "\x00" + canonicalDiagnosticTitleID(titleID)
	sum := sha256.Sum256([]byte(source))
	return "MPD-" + strings.ToUpper(hex.EncodeToString(sum[:8]))
}

// EffectiveGroupID recalculates the key even for older reports that lack the
// field, or imported reports containing a stale or manually edited key.
func (r DiagnosticReport) EffectiveGroupID() string {
	return DiagnosticGroupID(r.ErrorCode, r.Operation, r.TitleID)
}

// GroupDiagnosticReports accepts newest-first reports and orders the result by
// most recently observed group. Stable order is retained for equal timestamps.
// No reports are deleted or coalesced on disk; original report IDs survive.
func GroupDiagnosticReports(reports []DiagnosticReport) []DiagnosticGroup {
	ordered := append([]DiagnosticReport(nil), reports...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].OccurredUTC > ordered[j].OccurredUTC
	})
	groups := make([]DiagnosticGroup, 0)
	positions := make(map[string]int)
	for _, r := range ordered {
		if r.Schema != "magicpatchdesk-diagnostic-v1" {
			continue
		}
		id := r.EffectiveGroupID()
		i, ok := positions[id]
		if !ok {
			i = len(groups)
			positions[id] = i
			groups = append(groups, DiagnosticGroup{
				ID: id, Operation: diagnosticTag(r.Operation), ErrorCode: diagnosticTag(r.ErrorCode),
				TitleID: canonicalDiagnosticTitleID(r.TitleID), Latest: r,
				FirstOccurredUTC: r.OccurredUTC, LastOccurredUTC: r.OccurredUTC,
			})
		}
		g := &groups[i]
		g.Occurrences++
		if r.Severity == "warning" {
			g.Warnings++
		} else {
			g.Errors++
		}
		if r.OccurredUTC < g.FirstOccurredUTC {
			g.FirstOccurredUTC = r.OccurredUTC
		}
	}
	return groups
}

func (g DiagnosticGroup) IssueTitle() string {
	titleID := g.TitleID
	if titleID == "" {
		titleID = "UNKNOWN"
	}
	return fmt.Sprintf("[Diagnostic %s] %s / %s / %s", g.ID, g.ErrorCode, g.Operation, titleID)
}

func (g DiagnosticGroup) Summary() string {
	return fmt.Sprintf("Diagnostic group: %s\nCategory: %s\nOperation: %s\nTitle ID: %s\nLocal occurrences: %d (%d errors, %d warnings)\nFirst observed (UTC): %s\nLast observed (UTC): %s\n\nLatest local report:\n%s",
		g.ID, g.ErrorCode, g.Operation, g.TitleID, g.Occurrences,
		g.Errors, g.Warnings, g.FirstOccurredUTC, g.LastOccurredUTC, g.Latest.Summary())
}

// RecentDiagnosticGroups only reads local files. It never contacts GitHub or
// transmits telemetry. Up to 1000 newest local reports are grouped.
func RecentDiagnosticGroups(root string) ([]DiagnosticGroup, error) {
	reports, err := RecentFailureReports(root, 1000)
	if err != nil {
		return nil, err
	}
	return GroupDiagnosticReports(reports), nil
}
