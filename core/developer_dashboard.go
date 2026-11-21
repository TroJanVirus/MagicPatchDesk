package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DashboardIssue contains only metadata voluntarily published in GitHub Issues.
// No user file, game data or local diagnostic report is uploaded when fetching.
type DashboardIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest *json.RawMessage `json:"pull_request,omitempty"`
}

// DashboardFetchResult is a read-only snapshot; Incomplete is true when the
// GitHub API's pagination cap is reached (do not assert complete coverage).
type DashboardFetchResult struct {
	Issues     []DashboardIssue
	FetchedAt  time.Time
	Incomplete bool
}

const dashboardMaxPages = 5 // 500 entries max; conserve rate-limited anonymous API calls.

// FetchDeveloperIssues requests ONLY public issues for the fixed project.
// Call this only after the user explicitly presses Refresh GitHub. No token is
// stored, no write endpoint is used, and no diagnostic is transmitted.
func FetchDeveloperIssues(ctx context.Context, client *http.Client) (DashboardFetchResult, error) {
	return fetchDeveloperIssues(ctx, client, "https://api.github.com/repos/TroJanVirus/MagicPatchDesk/issues")
}

// An injected URL makes the network behaviour testable without GitHub.
func fetchDeveloperIssues(ctx context.Context, client *http.Client, endpoint string) (DashboardFetchResult, error) {
	var result DashboardFetchResult
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	// Keep the trust boundary explicit even when testing with an injected URL.
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return result, errors.New("invalid issues endpoint")
	}
	seen := map[int]bool{}
	for page := 1; page <= dashboardMaxPages; page++ {
		u := *parsed
		q := u.Query()
		q.Set("state", "all")
		q.Set("per_page", "100")
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return result, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		req.Header.Set("User-Agent", "MagicPatchDesk-Diagnostics/0.10")
		resp, err := client.Do(req)
		if err != nil {
			return result, fmt.Errorf("GitHub issues fetch failed: %w", err)
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
		resp.Body.Close()
		if readErr != nil {
			return result, fmt.Errorf("read GitHub response: %w", readErr)
		}
		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
				return result, errors.New("GitHub rate limit or access restriction; use Open GitHub Issues instead")
			}
			return result, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
		}
		if len(b) > 4<<20 {
			return result, errors.New("GitHub response exceeds safe size limit")
		}
		var items []DashboardIssue
		if err := json.Unmarshal(b, &items); err != nil {
			return result, fmt.Errorf("invalid GitHub issues response: %w", err)
		}
		for _, issue := range items {
			if issue.Number <= 0 || issue.Number > 1000000000 || seen[issue.Number] {
				continue
			}
			seen[issue.Number] = true
			// Keep PRs in the snapshot for showing a referenced fix PR's
			// open/closed state, but exclude them from diagnostic issue counts.
			result.Issues = append(result.Issues, issue)
		}
		if len(items) < 100 {
			break
		}
		if page == dashboardMaxPages {
			result.Incomplete = true
		}
	}
	result.FetchedAt = time.Now().UTC()
	return result, nil
}

var diagnosticGroupMarkerRE = regexp.MustCompile(`(?i)\bMPD-[0-9a-f]{16}\b`)
var diagnosticReportTitleRE = regexp.MustCompile(`(?i)^\s*\[Diagnostic\s+MPD-[0-9a-f]{16}\]`)
var dashboardRootMarkerRE = regexp.MustCompile(`(?im)^\s*Root cause\s*:\s*#(\d{1,9})\s*$`)
var dashboardFixMarkerRE = regexp.MustCompile(`(?im)^\s*Fix PR\s*:\s*#(\d{1,9})\s*$`)

func parseIssueGroupIDs(issue DashboardIssue) (ids []string, isReport bool) {
	found := map[string]bool{}
	add := func(s string) {
		for _, m := range diagnosticGroupMarkerRE.FindAllString(s, -1) {
			v := strings.ToUpper(m)
			if !found[v] {
				ids = append(ids, v)
				found[v] = true
			}
		}
	}
	// A report is identifiable by title or by the existing diagnostic group's
	// fixed-format body. Other issue bodies aren't arbitrary group references.
	isReport = diagnosticReportTitleRE.MatchString(issue.Title)
	if isReport {
		add(issue.Title)
	}
	for _, line := range strings.Split(issue.Body, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "diagnostic group:") || strings.HasPrefix(lower, "group:") {
			isReport = true
			add(line)
		} else if strings.HasPrefix(lower, "related groups:") || strings.HasPrefix(lower, "diagnostic groups:") {
			add(line)
		}
	}
	return ids, isReport
}
func firstIssueNumber(re *regexp.Regexp, body string) int {
	match := re.FindStringSubmatch(body)
	if len(match) != 2 {
		return 0
	}
	n, e := strconv.Atoi(match[1])
	if e != nil || n < 1 || n > 1000000000 {
		return 0
	}
	return n
}

// DeveloperCase is one underlying problem. Multiple diagnostic group IDs can
// share a case when their GitHub issues explicitly identify the same root
// cause. Conflicting root-cause claims remain separate and are flagged.
type DeveloperCase struct {
	Key                string
	GroupIDs           []string
	RootIssue          int
	RootIssueState     string
	FixPR              int
	FixPRState         string
	LinkedIssueNumbers []int
	LocalOccurrences   int
	GitHubReports      int // Number of distinct report issues; not GitHub comments or people.
	OpenReports        int
	ClosedReports      int
	Category           string
	Operation          string
	TitleID            string
	LatestUTC          string
	RootConflict       bool
}

func (c DeveloperCase) Label() string {
	if c.RootIssue > 0 {
		return fmt.Sprintf("Root cause #%d", c.RootIssue)
	}
	if len(c.GroupIDs) > 0 {
		return c.GroupIDs[0]
	}
	return "Unclassified"
}
func (c DeveloperCase) Summary() string {
	root := "No root-cause issue linked"
	if c.RootIssue > 0 {
		root = fmt.Sprintf("Root cause #%d (%s)", c.RootIssue, c.RootIssueState)
	}
	fix := "No fix PR linked"
	if c.FixPR > 0 {
		fix = fmt.Sprintf("Fix PR #%d (%s)", c.FixPR, c.FixPRState)
	}
	conflict := ""
	if c.RootConflict {
		conflict = "\nWarning: conflicting root-cause claims; do not auto-merge."
	}
	return fmt.Sprintf("Magic Patch Desk developer case\n%s\nGroups: %s\nCategory: %s\nOperation: %s\nTitle ID: %s\nGitHub report issues: %d (%d open, %d closed)\nLocal occurrences on this PC: %d\n%s\n%s\nRelated issue numbers: %v%s\n\nNo game data or private local reports have been transmitted.",
		c.Label(), strings.Join(c.GroupIDs, ", "), c.Category, c.Operation, c.TitleID,
		c.GitHubReports, c.OpenReports, c.ClosedReports, c.LocalOccurrences, root, fix, c.LinkedIssueNumbers, conflict)
}

// DeveloperSort determines the dashboard order and is a UI setting only.
type DeveloperSort int

const (
	SortByReports DeveloperSort = iota
	SortByNewest
	SortOpenFirst
)

// BuildDeveloperCases groups submitted diagnostic issues with local failure
// reports. Root cause markers are user-authored GitHub metadata, not proof of
// shared causation. No write, network call or on-disk modification occurs.
func BuildDeveloperCases(local []DiagnosticReport, issues []DashboardIssue, mode DeveloperSort) []DeveloperCase {
	type groupInfo struct {
		id, category, operation, titleID, latest string
		local, reportCount, open, closed         int
		issueIDs                                 map[int]bool
		roots                                    map[int]bool
		fixPR                                    int
	}
	groups := map[string]*groupInfo{}
	get := func(id string) *groupInfo {
		if groups[id] == nil {
			groups[id] = &groupInfo{id: id, issueIDs: map[int]bool{}, roots: map[int]bool{}, titleID: "UNKNOWN", category: "unknown", operation: "unknown"}
		}
		return groups[id]
	}
	for _, g := range GroupDiagnosticReports(local) {
		v := get(g.ID)
		v.local = g.Occurrences
		v.category = g.ErrorCode
		v.operation = g.Operation
		v.titleID = g.TitleID
		v.latest = g.LastOccurredUTC
	}
	issueByNumber := map[int]DashboardIssue{}
	for _, issue := range issues {
		if issue.Number > 0 && issue.Number <= 1000000000 {
			issueByNumber[issue.Number] = issue
		}
	}
	for _, issue := range issues {
		if issue.Number <= 0 || issue.Number > 1000000000 || issue.PullRequest != nil {
			continue
		}
		ids, isReport := parseIssueGroupIDs(issue)
		if len(ids) == 0 {
			continue
		}
		root := firstIssueNumber(dashboardRootMarkerRE, issue.Body)
		fix := firstIssueNumber(dashboardFixMarkerRE, issue.Body)
		for _, id := range ids {
			g := get(id)
			g.issueIDs[issue.Number] = true
			if isReport {
				g.reportCount++
				if strings.EqualFold(issue.State, "open") {
					g.open++
				} else {
					g.closed++
				}
			}
			if root > 0 {
				g.roots[root] = true
			} else if !isReport {
				g.roots[issue.Number] = true
			}
			if fix > 0 && (g.fixPR == 0 || g.fixPR == fix) {
				g.fixPR = fix
			}
			if updated := issue.UpdatedAt.UTC().Format(time.RFC3339); updated > g.latest {
				g.latest = updated
			}
		}
	}
	cases := map[string]*DeveloperCase{}
	for _, g := range groups {
		root, conflict := 0, false
		for n := range g.roots {
			if root == 0 {
				root = n
			} else if n != root {
				conflict = true
			}
		}
		if conflict {
			root = 0
		}
		key := g.id
		if root != 0 {
			key = fmt.Sprintf("ROOT-%d", root)
		}
		c := cases[key]
		if c == nil {
			c = &DeveloperCase{Key: key, RootIssue: root, TitleID: g.titleID, Category: g.category, Operation: g.operation, RootConflict: conflict}
			cases[key] = c
		}
		c.GroupIDs = append(c.GroupIDs, g.id)
		c.LocalOccurrences += g.local
		c.GitHubReports += g.reportCount
		c.OpenReports += g.open
		c.ClosedReports += g.closed
		c.RootConflict = c.RootConflict || conflict
		if g.latest > c.LatestUTC {
			c.LatestUTC = g.latest
		}
		if c.FixPR == 0 {
			c.FixPR = g.fixPR
		} else if g.fixPR != 0 && g.fixPR != c.FixPR {
			c.FixPR = 0
			c.RootConflict = true
		}
		for id := range g.issueIDs {
			c.LinkedIssueNumbers = append(c.LinkedIssueNumbers, id)
		}
	}
	out := make([]DeveloperCase, 0, len(cases))
	for _, c := range cases {
		sort.Strings(c.GroupIDs)
		sort.Ints(c.LinkedIssueNumbers)
		c.LinkedIssueNumbers = uniqueInts(c.LinkedIssueNumbers)
		if len(c.GroupIDs) > 1 {
			c.Category = "multiple"
			c.Operation = "multiple"
			c.TitleID = "MULTIPLE"
		}
		if r, ok := issueByNumber[c.RootIssue]; ok {
			c.RootIssueState = strings.ToLower(r.State)
		} else if c.RootIssue > 0 {
			c.RootIssueState = "not in fetched issue pages"
		}
		if p, ok := issueByNumber[c.FixPR]; ok && p.PullRequest != nil {
			c.FixPRState = strings.ToLower(p.State)
		} else if c.FixPR > 0 {
			c.FixPRState = "not verified"
		}
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch mode {
		case SortByNewest:
			if a.LatestUTC != b.LatestUTC {
				return a.LatestUTC > b.LatestUTC
			}
		case SortOpenFirst:
			if a.OpenReports != b.OpenReports {
				return a.OpenReports > b.OpenReports
			}
		default:
			if a.GitHubReports != b.GitHubReports {
				return a.GitHubReports > b.GitHubReports
			}
			if a.LocalOccurrences != b.LocalOccurrences {
				return a.LocalOccurrences > b.LocalOccurrences
			}
		}
		if a.GitHubReports != b.GitHubReports {
			return a.GitHubReports > b.GitHubReports
		}
		if a.LocalOccurrences != b.LocalOccurrences {
			return a.LocalOccurrences > b.LocalOccurrences
		}
		return a.Key < b.Key
	})
	return out
}
func uniqueInts(xs []int) []int {
	if len(xs) < 2 {
		return xs
	}
	w := 1
	for i := 1; i < len(xs); i++ {
		if xs[i] != xs[w-1] {
			xs[w] = xs[i]
			w++
		}
	}
	return xs[:w]
}

func DeveloperIssueURL(number int) string {
	if number <= 0 || number > 1000000000 {
		return ""
	}
	return fmt.Sprintf("https://github.com/TroJanVirus/MagicPatchDesk/issues/%d", number)
}
func DeveloperPRURL(number int) string {
	if number <= 0 || number > 1000000000 {
		return ""
	}
	return fmt.Sprintf("https://github.com/TroJanVirus/MagicPatchDesk/pull/%d", number)
}
func (c DeveloperCase) RelatedGroupsMarker() string {
	return "Related groups: " + strings.Join(c.GroupIDs, ", ")
}
func (c DeveloperCase) RootCauseMarker() string {
	if c.RootIssue <= 0 {
		return ""
	}
	return fmt.Sprintf("Root cause: #%d", c.RootIssue)
}
