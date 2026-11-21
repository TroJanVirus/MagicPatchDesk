package core

import (
	"errors"
	"strings"
	"testing"
)

func TestDiagnosticGroupingStableAcrossReportsAndSessions(t *testing.T) {
	ctx := FailureContext{Operation: "patch.apply", TitleID: "4a580001", PatchPath: "my-private.jmp", SourcePath: "C:/Personal Stuff/default.xbe"}
	a, err := NewFailureReport(ctx, errors.New("CRC32 mismatch"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewFailureReport(ctx, errors.New("checksum mismatch"))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatal("individual report IDs must differ")
	}
	if a.GroupID != b.GroupID || !strings.HasPrefix(a.GroupID, "MPD-") {
		t.Fatal("same issue should group", a.GroupID, b.GroupID)
	}
	c, err := NewFailureReport(FailureContext{Operation: "patch.convert", TitleID: "4a580001"}, errors.New("CRC32 mismatch"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewFailureReport(FailureContext{Operation: "patch.apply", TitleID: "4343000E"}, errors.New("CRC32 mismatch"))
	if err != nil {
		t.Fatal(err)
	}
	if c.GroupID == a.GroupID || d.GroupID == a.GroupID {
		t.Fatal("different operation or TitleID must separate")
	}
	for _, r := range []DiagnosticReport{a, b, c, d} {
		if strings.Contains(r.Summary(), "Personal Stuff") || strings.Contains(r.Summary(), "my-private.jmp") {
			t.Fatal("private path leaked")
		}
		if !strings.Contains(r.Summary(), r.GroupID) || !strings.Contains(r.Summary(), "Title ID:") {
			t.Fatal("missing useful identifier")
		}
	}
	// Legacy reports with no group_id are grouped using the same canonical key.
	b.GroupID = ""
	a.OccurredUTC = "2026-11-21T10:00:00Z"
	b.OccurredUTC = "2026-11-21T11:00:00Z"
	c.OccurredUTC = "2026-11-21T09:00:00Z"
	d.OccurredUTC = "2026-11-21T08:00:00Z"
	g := GroupDiagnosticReports([]DiagnosticReport{c, d, a, b})
	if len(g) != 3 || g[0].Occurrences != 2 || g[0].Latest.ID != b.ID {
		t.Fatalf("incorrect grouping %#v", g)
	}
	if g[0].Errors != 2 || g[0].Warnings != 0 {
		t.Fatalf("bad severity counts %#v", g[0])
	}
	if !strings.Contains(g[0].IssueTitle(), a.GroupID) {
		t.Fatal("issue title cannot be searched by group")
	}
}

func TestGroupIsolationAndLocalPersistence(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 4; i++ {
		op := "pf.apply"
		if i == 3 {
			op = "pack.apply"
		}
		r, e := NewFailureReport(FailureContext{Operation: op, TitleID: "00000001"}, errors.New("pattern not found"))
		if e != nil {
			t.Fatal(e)
		}
		if _, e = SaveFailureReport(root, r); e != nil {
			t.Fatal(e)
		}
	}
	groups, e := RecentDiagnosticGroups(root)
	if e != nil || len(groups) != 2 {
		t.Fatalf("local group failure %v %#v", e, groups)
	}
	var total int
	for _, g := range groups {
		total += g.Occurrences
	}
	if total != 4 {
		t.Fatalf("lost reports: %d", total)
	}
	if len(GroupDiagnosticReports(nil)) != 0 {
		t.Fatal("empty group should stay empty")
	}
	if DiagnosticGroupID("checksum_mismatch", "pf.apply", "PRIVATE-WORLD") != DiagnosticGroupID("checksum_mismatch", "pf.apply", "") {
		t.Fatal("invalid TitleID must not enter key")
	}
	if DiagnosticGroupID("checksum_mismatch", "pf.apply", "00000001") == DiagnosticGroupID("checksum_mismatch", "pf.apply", "00000002") {
		t.Fatal("Title ID collision")
	}
}
