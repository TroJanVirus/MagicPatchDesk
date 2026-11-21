package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DiagnosticReport deliberately contains no raw patches, game bytes, passwords,
// environment variables, username, absolute paths, or unfiltered error strings.
// Reports are local-only until the user explicitly chooses to share one.
type DiagnosticReport struct {
	Schema          string `json:"schema"`
	ID              string `json:"id"`
	GroupID         string `json:"group_id,omitempty"`
	OccurredUTC     string `json:"occurred_utc"`
	AppVersion      string `json:"app_version"`
	OperatingSystem string `json:"os"`
	Architecture    string `json:"arch"`
	Operation       string `json:"operation"`
	Severity        string `json:"severity"`
	ErrorCode       string `json:"error_code"`
	WarningCount    int    `json:"warning_count,omitempty"`
	Explanation     string `json:"explanation"`
	PatchFormat     string `json:"patch_format,omitempty"`
	OutputFormat    string `json:"output_format,omitempty"`
	TargetExtension string `json:"target_extension,omitempty"`
	SourceKind      string `json:"source_kind,omitempty"`
	TitleID         string `json:"title_id,omitempty"`
	SourceCRC32     string `json:"source_crc32,omitempty"`
	ExpectedCRC32   string `json:"expected_crc32,omitempty"`
	RecordNumber    int    `json:"record_number,omitempty"`
	ByteOffset      string `json:"byte_offset,omitempty"`
}

// FailureContext includes only known-safe structured metadata; filenames are
// deliberately converted to their extension instead of being recorded.
type FailureContext struct {
	AppVersion   string
	Operation    string
	PatchPath    string
	SourcePath   string
	OutputFormat string
	TitleID      string
	SourceCRC32  string
	ExpectedCRC  string
	WarningCount int
}

var diagnosticRecordRE = regexp.MustCompile(`(?i)\brecord\s+(\d{1,8})\b`)
var diagnosticOffsetRE = regexp.MustCompile(`(?i)\b(?:offset|at)\s+(0x[0-9a-f]{1,12})\b`)
var diagnosticTitleIDRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}$`)
var diagnosticCRC32RE = regexp.MustCompile(`(?i)^[0-9a-f]{8}$`)

func diagnosticTag(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
		if b.Len() >= 50 {
			break
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func knownFileType(s string) string {
	if s == "" {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(s))
	switch ext {
	case ".xbe", ".iso", ".xiso", ".cci", ".cfg", ".ini", ".bin", ".dat", ".pak", ".img", ".jmp", ".pf", ".ips", ".eips", ".bps", ".xdelta", ".vcdiff", ".mpdpack":
		return ext
	default:
		return "other" // Do not disclose arbitrary filename parts.
	}
}

func categorizeFailure(e error) (string, string) {
	if e == nil {
		return "none", "No failure was supplied."
	}
	s := strings.ToLower(e.Error())
	for _, x := range []struct {
		fragments         []string
		code, description string
	}{
		{[]string{"crc32", "checksum mismatch", "sha256", "sha-256", "hash mismatch"}, "checksum_mismatch", "An input or output checksum did not match the expected value."},
		{[]string{"title id mismatch", "titleid mismatch", "wrong title id"}, "title_id_mismatch", "The selected game has a different Title ID than the patch requires."},
		{[]string{"pattern not found", "search pattern not found", "hex pattern not found"}, "search_not_found", "A search-and-replace pattern was not present in the target."},
		{[]string{"missing source", "original file", "requires the source", "requires original", "source file required"}, "missing_original", "The conversion or patch requires original input bytes."},
		{[]string{"ambiguous", "multiple matches", "multiple possible"}, "ambiguous_target", "More than one possible target file matched."},
		{[]string{"path traversal", "unsafe path", "unsafe target", "reserved device"}, "unsafe_target", "An unsafe or invalid target path was rejected."},
		{[]string{"unsupported", "cannot represent", "not supported", "invalid format", "full-disc iso output requires", "only xbe output is possible", "would be incomplete"}, "unsupported_format", "The requested operation is not supported by that format."},
		{[]string{"outside", "out of bounds", "offset", "exceeds", "too large", "sector"}, "out_of_bounds", "The patch refers to unavailable space or exceeds a file-size safety limit."},
		{[]string{"permission denied", "access is denied"}, "access_denied", "Windows denied access to a source or output location."},
		{[]string{"not exist", "not found", "no such file"}, "missing_file", "A required file could not be located."},
		{[]string{"cancelled", "canceled"}, "cancelled", "The operation was cancelled."},
		{[]string{"unexpected eof", "malformed", "corrupt", "invalid patch", "truncated"}, "invalid_patch", "The patch data could not be decoded or validated."},
	} {
		for _, phrase := range x.fragments {
			if strings.Contains(s, phrase) {
				return x.code, x.description
			}
		}
	}
	return "operation_failed", "The operation failed. Sensitive raw error text is not included in shared reports."
}

// NewFailureReport stores only allowlisted metadata and classified failure
// information. Err.Error() is never copied into the report verbatim.
func NewFailureReport(ctx FailureContext, failure error) (DiagnosticReport, error) {
	if failure == nil {
		return DiagnosticReport{}, errors.New("diagnostic report requires a failure")
	}
	var nonce [9]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return DiagnosticReport{}, err
	}
	code, msg := categorizeFailure(failure)
	severity := "error"
	if strings.HasSuffix(ctx.Operation, ".warning") {
		severity = "warning"
		code = "operation_warning"
		msg = "One or more patch operations were skipped or produced warnings. Review the source patch and target before distribution."
	}
	report := DiagnosticReport{
		Schema: "magicpatchdesk-diagnostic-v1", ID: hex.EncodeToString(nonce[:]),
		OccurredUTC: time.Now().UTC().Format(time.RFC3339), AppVersion: diagnosticTag(ctx.AppVersion),
		OperatingSystem: runtime.GOOS, Architecture: runtime.GOARCH,
		Operation: diagnosticTag(ctx.Operation), Severity: severity, WarningCount: ctx.WarningCount, ErrorCode: code, Explanation: msg,
		PatchFormat: knownFileType(ctx.PatchPath), OutputFormat: knownFileType(ctx.OutputFormat),
		TargetExtension: knownFileType(ctx.SourcePath),
	}
	switch report.TargetExtension {
	case ".iso", ".xiso":
		report.SourceKind = "disc-image"
    case ".cci":
        report.SourceKind = "cerbios-cci"
	case ".xbe":
		report.SourceKind = "loose-executable"
	case "":
		report.SourceKind = "unspecified"
	default:
		report.SourceKind = "other-file"
	}
	if diagnosticTitleIDRE.MatchString(ctx.TitleID) {
		report.TitleID = strings.ToUpper(ctx.TitleID)
	}
	if diagnosticCRC32RE.MatchString(ctx.SourceCRC32) {
		report.SourceCRC32 = strings.ToUpper(ctx.SourceCRC32)
	}
	if diagnosticCRC32RE.MatchString(ctx.ExpectedCRC) {
		report.ExpectedCRC32 = strings.ToUpper(ctx.ExpectedCRC)
	}
	if match := diagnosticRecordRE.FindStringSubmatch(failure.Error()); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			report.RecordNumber = n
		}
	}
	if match := diagnosticOffsetRE.FindStringSubmatch(failure.Error()); match != nil {
		report.ByteOffset = strings.ToUpper(match[1])
	}
	report.GroupID = report.EffectiveGroupID()
	return report, nil
}

func (r DiagnosticReport) Summary() string {
	title := canonicalDiagnosticTitleID(r.TitleID)
	return fmt.Sprintf("Magic Patch Desk %s | report %s\nGroup: %s\nSeverity: %s\nCategory: %s\nOperation: %s\nTitle ID: %s\nResult: %s\nInput: %s / %s\nTarget: %s\nOS: %s/%s\nRecord: %d\nWarnings: %d\nNo game content, local paths, credentials or raw error strings are included.",
		r.AppVersion, r.ID, r.EffectiveGroupID(), r.Severity,
		diagnosticTag(r.ErrorCode), diagnosticTag(r.Operation), title,
		r.Explanation, r.PatchFormat, r.SourceKind, r.TargetExtension,
		r.OperatingSystem, r.Architecture, r.RecordNumber, r.WarningCount)
}

// SaveFailureReport creates a private JSON report, never overwriting a prior
// one. Reports are kept only in the user's local app-data Reports directory.
func SaveFailureReport(root string, report DiagnosticReport) (string, error) {
	if report.Schema != "magicpatchdesk-diagnostic-v1" || !regexp.MustCompile(`^[a-f0-9]{18}$`).MatchString(report.ID) {
		return "", errors.New("invalid diagnostic report")
	}
	folder := filepath.Join(root, "Reports")
	if err := os.MkdirAll(folder, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(folder, "failure-"+report.ID+".json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if len(data) > 8192 {
		return "", errors.New("diagnostic metadata exceeds size limit")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := f.Write(append(data, '\n'))
	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(path)
		return "", writeErr
	}
	if closeErr != nil {
		os.Remove(path)
		return "", closeErr
	}
	return path, nil
}

func RecentFailureReports(root string, maxCount int) ([]DiagnosticReport, error) {
	if maxCount < 1 || maxCount > 1000 {
		return nil, errors.New("invalid diagnostic report limit")
	}
	folder := filepath.Join(root, "Reports")
	entries, err := os.ReadDir(folder)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "failure-") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool {
		a, _ := os.Stat(filepath.Join(folder, names[i]))
		b, _ := os.Stat(filepath.Join(folder, names[j]))
		if a == nil || b == nil {
			return names[i] > names[j]
		}
		return a.ModTime().After(b.ModTime())
	})
	results := make([]DiagnosticReport, 0, maxCount)
	for _, name := range names {
		if len(results) >= maxCount {
			break
		}
		path := filepath.Join(folder, name)
		fileInfo, statErr := os.Stat(path)
		if statErr != nil || fileInfo.Size() > 8192 || !fileInfo.Mode().IsRegular() {
			continue
		}
		bytes, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		var r DiagnosticReport
		if json.Unmarshal(bytes, &r) == nil && r.Schema == "magicpatchdesk-diagnostic-v1" {
			results = append(results, r)
		}
	}
	return results, nil
}
