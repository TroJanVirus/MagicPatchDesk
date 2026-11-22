package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const GitHubIssuesAPI = "https://api.github.com/repos/" + ReleaseRepo

var safeIssueID = regexp.MustCompile(`^MPD-[A-F0-9]{16}$`)
var safeReportID = regexp.MustCompile(`^[a-f0-9]{18}$`)

type ReportUploadState struct {
	Schema    string         `json:"schema"`
	Submitted map[string]int `json:"submitted"` // Report IDs -> GitHub issue numbers.
}

func reportSyncPath(root string) string {
	return filepath.Join(root, "Reports", "submission-state.json")
}
func LoadReportUploadState(root string) (ReportUploadState, error) {
	result := ReportUploadState{Schema: "magicpatchdesk-report-sync-v1", Submitted: map[string]int{}}
	b, err := os.ReadFile(reportSyncPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if len(b) > 64<<10 {
		return result, errors.New("report sync state exceeds size limit")
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return result, err
	}
	if result.Schema != "magicpatchdesk-report-sync-v1" || result.Submitted == nil {
		return result, errors.New("invalid report upload state")
	}
	for id, n := range result.Submitted {
		if !safeReportID.MatchString(id) || n < 1 {
			delete(result.Submitted, id)
		}
	}
	return result, nil
}
func SaveReportUploadState(root string, v ReportUploadState) error {
	if v.Submitted == nil {
		v.Submitted = map[string]int{}
	}
	if len(v.Submitted) > 600 { // Keep only bounded metadata; oldest IDs can be safely re-tried after pruning.
		keys := make([]string, 0, len(v.Submitted))
		for k := range v.Submitted {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys[:len(keys)-600] {
			delete(v.Submitted, k)
		}
	}
	v.Schema = "magicpatchdesk-report-sync-v1"
	dir := filepath.Dir(reportSyncPath(root))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".report-sync-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), reportSyncPath(root))
}
func IssueReportBody(r DiagnosticReport) string {
	id := r.EffectiveGroupID()
	return "### Automatically generated, privacy-filtered diagnostic\n\n" +
		"Group: " + id + "\nTitle ID: " + canonicalDiagnosticTitleID(r.TitleID) + "\n\n" + r.Summary() + "\n\n" +
		"Report ID: " + r.ID + "\n\n" +
		"This report contains no binary game data, local file paths, patch payloads or passwords.\n" +
		"Please add reproduction steps if available.\n"
}

// sanitizeReportForIssue accepts only fixed schema metadata and replaces the
// explanation with known text. This protects against manually edited/imported
// local JSON containing accidental filenames, passwords or arbitrary content.
func sanitizeReportForIssue(r DiagnosticReport) DiagnosticReport {
	safe := DiagnosticReport{Schema: "magicpatchdesk-diagnostic-v1", ID: r.ID, AppVersion: diagnosticTag(r.AppVersion),
		Operation: diagnosticTag(r.Operation), ErrorCode: diagnosticTag(r.ErrorCode),
		OperatingSystem: "windows", Architecture: "amd64",
		Severity: "error"}
	if r.Severity == "warning" {
		safe.Severity = "warning"
	}
	messages := map[string]string{
		"checksum_mismatch":  "The source or result did not match its expected checksum.",
		"title_id_mismatch":  "The selected game has a different Title ID.",
		"search_not_found":   "A patch search pattern was not present.",
		"missing_original":   "This operation requires original file bytes.",
		"ambiguous_target":   "Multiple possible target files were found.",
		"unsafe_target":      "An unsafe or invalid target path was rejected.",
		"unsupported_format": "The selected format cannot represent this operation.",
		"out_of_bounds":      "The operation exceeded safe bounds.",
		"access_denied":      "Operating system denied access to a file.",
		"missing_file":       "A required file was missing.",
		"cancelled":          "The operation was cancelled.",
		"invalid_patch":      "The patch could not be decoded safely.",
		"operation_warning":  "The operation reported warnings or skipped an edit.",
		"operation_failed":   "The operation failed; sensitive error text is excluded.",
        "unhandled_panic": "The app encountered an unhandled panic; private stack trace remains on the user device.",
        "unexpected_exit": "The previous app process did not shut down cleanly. Possible crash, forced exit or power interruption.",
	}
	safe.Explanation = messages[safe.ErrorCode]
	if safe.Explanation == "" {
		safe.ErrorCode = "operation_failed"
		safe.Explanation = messages[safe.ErrorCode]
	}
	if diagnosticTitleIDRE.MatchString(r.TitleID) {
		safe.TitleID = strings.ToUpper(r.TitleID)
	}
	if r.WarningCount >= 0 && r.WarningCount <= 100000 {
		safe.WarningCount = r.WarningCount
	}
	if r.RecordNumber >= 0 && r.RecordNumber <= 100000000 {
		safe.RecordNumber = r.RecordNumber
	}
	allowedExt := map[string]bool{".xbe": true, ".iso": true, ".xiso": true, ".cci": true, ".pf": true, ".jmp": true, ".bps": true, ".ips": true, ".eips": true, ".xdelta": true, ".vcdiff": true, ".mpdpack": true, "other": true}
	if allowedExt[r.PatchFormat] {
		safe.PatchFormat = r.PatchFormat
	}
	if allowedExt[r.TargetExtension] {
		safe.TargetExtension = r.TargetExtension
	}
	if allowedExt[r.OutputFormat] {
		safe.OutputFormat = r.OutputFormat
	}
	switch r.SourceKind {
	case "disc-image", "loose-executable", "other-file", "unspecified":
		safe.SourceKind = r.SourceKind
	default:
		safe.SourceKind = "unspecified"
	}
	return safe
}

// PublishDiagnostic creates one GitHub issue per stable failure group and adds
// subsequent events as comments. It is only called after users explicitly opt
// in and provide their own GitHub credentials. No token is stored or logged.
func PublishDiagnostic(ctx context.Context, client *http.Client, base, token string, r DiagnosticReport) (int, error) {
	if !safeReportID.MatchString(r.ID) || r.Schema != "magicpatchdesk-diagnostic-v1" {
		return 0, errors.New("invalid report ID")
	}
	r = sanitizeReportForIssue(r)
	group := r.EffectiveGroupID()
	if !safeIssueID.MatchString(group) {
		return 0, errors.New("invalid group ID")
	}
	if strings.TrimSpace(token) == "" {
		return 0, errors.New("GitHub credential is required")
	}
	if base == "" {
		base = GitHubIssuesAPI
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	searchURL := strings.TrimSuffix(base, "/issues") // allow mock API base matching the repo endpoint.
	var result struct {
		Items []struct {
			Number      int             `json:"number"`
			Title       string          `json:"title"`
			PullRequest json.RawMessage `json:"pull_request"`
		} `json:"items"`
	}
	query := url.Values{}
	query.Set("q", "repo:"+ReleaseRepo+" is:issue in:title "+group)
	parsedBase, err := url.Parse(searchURL)
	if err != nil {
		return 0, err
	}
	searchEndpoint := parsedBase.Scheme + "://" + parsedBase.Host + "/search/issues?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, "GET", searchEndpoint, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "MagicPatchDesk/"+AppVersion)
	resp, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return 0, fmt.Errorf("GitHub issue search HTTP %d", resp.StatusCode)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 200<<10)).Decode(&result)
	resp.Body.Close()
	if err != nil {
		return 0, err
	}
	number := 0
	for _, issue := range result.Items {
		if issue.Number > 0 && len(issue.PullRequest) == 0 && strings.Contains(issue.Title, group) {
			number = issue.Number
			break
		}
	}
	method := http.MethodPost
	route := searchURL + "/issues"
	title := "[Diagnostic " + group + "] " + diagnosticTag(r.ErrorCode) + " / " + diagnosticTag(r.Operation)
	payload := map[string]string{}
	if number == 0 {
		payload["title"] = title
		payload["body"] = IssueReportBody(r)
	} else {
		route = fmt.Sprintf("%s/issues/%d/comments", searchURL, number)
		payload["body"] = IssueReportBody(r)
	}
	body, _ := json.Marshal(payload)
	request, err = http.NewRequestWithContext(ctx, method, route, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "MagicPatchDesk/"+AppVersion)
	resp, err = client.Do(request)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return 0, fmt.Errorf("GitHub issue submission HTTP %d (report retained locally)", resp.StatusCode)
	}
	if number == 0 {
		var created struct {
			Number int `json:"number"`
		}
		if err = json.NewDecoder(io.LimitReader(resp.Body, 40<<10)).Decode(&created); err != nil {
			return 0, err
		}
		number = created.Number
	}
	if number <= 0 {
		return 0, errors.New("GitHub issue response missing number")
	}
	return number, nil
}

// SubmitQueuedReports sends locally redacted JSON reports, never game files.
// Only the most recent limit reports are attempted per call, preventing API
// floods; failures leave the queue intact for a later user-triggered retry.
func SubmitQueuedReports(ctx context.Context, root, token string, limit int) (sent int, err error) {
	if limit < 1 || limit > 10 {
		return 0, errors.New("invalid report batch limit")
	}
	state, err := LoadReportUploadState(root)
	if err != nil {
		return 0, err
	}
	reports, err := RecentFailureReports(root, 50)
	if err != nil {
		return 0, err
	}
	for _, r := range reports {
		if sent >= limit {
			break
		}
		if _, done := state.Submitted[r.ID]; done {
			continue
		}
		issue, e := PublishDiagnostic(ctx, nil, "", token, r)
		if e != nil {
			return sent, e
		}
		state.Submitted[r.ID] = issue
		if err = SaveReportUploadState(root, state); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
