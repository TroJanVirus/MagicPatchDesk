package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testDiagnostic(code, operation, title string) DiagnosticReport {
	r := DiagnosticReport{Schema: "magicpatchdesk-diagnostic-v1", ID: "0123456789abcdef01", OccurredUTC: "2026-11-21T04:00:00Z", Severity: "error", ErrorCode: code, Operation: operation, TitleID: title}
	r.GroupID = r.EffectiveGroupID()
	return r
}
func issue(n int, title, body, state string) DashboardIssue {
	return DashboardIssue{Number: n, Title: title, Body: body, State: state, UpdatedAt: time.Date(2026, 11, 21, 5+n%10, 0, 0, 0, time.UTC)}
}
func TestDeveloperDashboardGroupsAndLinks(t *testing.T) {
	a := testDiagnostic("checksum_mismatch", "patch.apply", "4A580001")
	b := testDiagnostic("missing_original", "patch.convert", "4A580001")
	root := issue(99, "[Root cause] wrong source selection", "Related groups: "+a.GroupID+", "+b.GroupID+"\nFix PR: #117", "open")
	report1 := issue(101, "[Diagnostic "+a.GroupID+"] checksum mismatch", "Root cause: #99", "open")
	report2 := issue(102, "[Diagnostic "+a.GroupID+"] repeat", "Root cause: #99", "closed")
	report3 := issue(103, "[Diagnostic "+b.GroupID+"] needs original", "Root cause: #99", "open")
	pr := issue(117, "Fix source resolver", "", "open")
	blob := json.RawMessage(`{}`)
	pr.PullRequest = &blob
	cases := BuildDeveloperCases([]DiagnosticReport{a, a, b}, []DashboardIssue{root, report1, report2, report3, pr}, SortByReports)
	if len(cases) != 1 {
		t.Fatalf("wanted one root case, got %+v", cases)
	}
	c := cases[0]
	if c.RootIssue != 99 || c.FixPR != 117 || c.FixPRState != "open" || c.RootIssueState != "open" {
		t.Errorf("tracking mismatch: %+v", c)
	}
	if c.LocalOccurrences != 3 || c.GitHubReports != 3 || c.OpenReports != 2 || c.ClosedReports != 1 {
		t.Errorf("counts wrong: %+v", c)
	}
	if len(c.GroupIDs) != 2 || len(c.LinkedIssueNumbers) != 4 {
		t.Errorf("links wrong: %+v", c)
	}
	if !strings.Contains(c.RelatedGroupsMarker(), a.GroupID) || !strings.Contains(c.RelatedGroupsMarker(), b.GroupID) {
		t.Errorf("unexpected marker %s", c.RelatedGroupsMarker())
	}
}
func TestDeveloperDashboardRootConflictsFailClosed(t *testing.T) {
	a := testDiagnostic("checksum_mismatch", "patch.apply", "4A580001")
	b := testDiagnostic("missing_original", "patch.convert", "4A580001")
	c := BuildDeveloperCases([]DiagnosticReport{a, b}, []DashboardIssue{
		issue(10, "[Diagnostic "+a.GroupID+"]", "Root cause: #70", "open"),
		issue(11, "[Diagnostic "+a.GroupID+"]", "Root cause: #80", "open"),
		issue(12, "[Diagnostic "+b.GroupID+"]", "Root cause: #70", "open"),
	}, SortByReports)
	if len(c) != 2 {
		t.Fatalf("conflicting roots must not merge: %#v", c)
	}
	conflicted := 0
	for _, v := range c {
		if v.RootConflict {
			conflicted++
			if v.RootIssue != 0 {
				t.Error("conflicting root retained")
			}
		}
	}
	if conflicted != 1 {
		t.Fatalf("expected one conflict: %+v", c)
	}
}
func TestDashboardFiltersPRsUnrelatedTextAndUnverifiedLinks(t *testing.T) {
	r := testDiagnostic("invalid_patch", "patch.preview", "4A580001")
	pr := issue(3, "[Diagnostic "+r.GroupID+"]", "", "open")
	blob := json.RawMessage(`{}`)
	pr.PullRequest = &blob
	other := issue(4, "Question", "Here is some prose about "+r.GroupID, "open")
	report := issue(7, "[Diagnostic "+r.GroupID+"]", "Fix PR: #51\nRoot cause: #42", "open")
	results := BuildDeveloperCases([]DiagnosticReport{r}, []DashboardIssue{pr, other, report}, SortByReports)
	if len(results) != 1 {
		t.Fatalf("unexpected cases: %+v", results)
	}
	c := results[0]
	if c.GitHubReports != 1 || c.RootIssue != 42 || c.RootIssueState != "not in fetched issue pages" || c.FixPRState != "not verified" {
		t.Fatalf("bad case: %+v", c)
	}
	if strings.Contains(c.Summary(), "private-path") {
		t.Fatal("leaked local path")
	}
	if DeveloperIssueURL(-1) != "" || DeveloperPRURL(1000000001) != "" {
		t.Fatal("unsafe issue URL")
	}
}
func TestDashboardSortOrders(t *testing.T) {
	a := testDiagnostic("checksum_mismatch", "patch.apply", "11111111")
	b := testDiagnostic("missing_original", "patch.convert", "22222222")
	issues := []DashboardIssue{
		issue(1, "[Diagnostic "+a.GroupID+"]", "", "closed"),
		issue(2, "[Diagnostic "+a.GroupID+"]", "", "closed"),
		issue(3, "[Diagnostic "+b.GroupID+"]", "", "open"),
	}
	reports := BuildDeveloperCases([]DiagnosticReport{a, b}, issues, SortByReports)
	if reports[0].GitHubReports != 2 {
		t.Fatal("not sorted by occurrence")
	}
	open := BuildDeveloperCases([]DiagnosticReport{a, b}, issues, SortOpenFirst)
	if open[0].GroupIDs[0] != b.GroupID {
		t.Fatal("not sorted open first")
	}
}
func TestFetchDeveloperIssuesPaginatesAndIsReadOnly(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Error("wrote to GitHub endpoint")
		}
		if r.URL.Query().Get("state") != "all" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("wrong filters: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		var items []DashboardIssue
		if r.URL.Query().Get("page") == "1" {
			for i := 1; i <= 100; i++ {
				items = append(items, issue(i, fmt.Sprintf("item%d", i), "", "open"))
			}
		} else {
			items = []DashboardIssue{issue(101, "final", "", "closed")}
		}
		if e := json.NewEncoder(w).Encode(items); e != nil {
			t.Error(e)
		}
	}))
	defer server.Close()
	result, err := fetchDeveloperIssues(context.Background(), server.Client(), server.URL)
	if err != nil || len(result.Issues) != 101 || calls != 2 || result.Incomplete {
		t.Fatalf("snapshot: %+v err %v calls %d", result, err, calls)
	}
	if result.FetchedAt.IsZero() {
		t.Fatal("no fetch timestamp")
	}
}
func TestFetchDeveloperIssuesRateLimitAndBadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer server.Close()
	_, err := fetchDeveloperIssues(context.Background(), server.Client(), server.URL)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "rate limit") {
		t.Fatalf("wanted clear error: %v", err)
	}
}
func TestDashboardNeverUsesRawLocalErrors(t *testing.T) {
	report := testDiagnostic("operation_failed", "pf.apply", "4A580001")
	report.Explanation = "This should never be surfaced in aggregated public issue titles: C:\\Users\\someone"
	c := BuildDeveloperCases([]DiagnosticReport{report}, nil, SortByReports)
	if len(c) != 1 || strings.Contains(c[0].Summary(), "someone") {
		t.Fatal("dashboard leaked local private detail")
	}
}
