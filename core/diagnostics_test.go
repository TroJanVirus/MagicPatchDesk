package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailureReportRedactsPathsAndSecrets(t *testing.T) {
	secret := "C:\\Users\\private-person\\Games\\Rare Game\\default.xbe"
	raw := "record 12 at 0xFF93: CRC32 mismatch at " + secret + " secret-password-XYZ"
	r, err := NewFailureReport(FailureContext{AppVersion: "v0.8", Operation: "patch.apply", SourcePath: secret, PatchPath: `C:\Users\private-person\Downloads\wide.jmp`, TitleID: "4a580001", SourceCRC32: "AB12CD34"}, errors.New(raw))
	if err != nil {
		t.Fatal(err)
	}
	if r.ErrorCode != "checksum_mismatch" || r.RecordNumber != 12 || r.ByteOffset != "0XFF93" {
		t.Fatalf("unexpected report %#v", r)
	}
	p, err := SaveFailureReport(t.TempDir(), r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"private-person", "Rare Game", "secret-password", "C:\\Users", "wide.jmp"} {
		if strings.Contains(string(data), s) {
			t.Fatalf("leaked sensitive data: %q", s)
		}
	}
	if !strings.Contains(string(data), "checksum_mismatch") {
		t.Fatal("failure category omitted")
	}
	if filepath.Base(p) != "failure-"+r.ID+".json" {
		t.Fatal("invalid stored report name")
	}
}
func TestFailureReportsLocalOnlyAndLimited(t *testing.T) {
	root := t.TempDir()
	r, e := NewFailureReport(FailureContext{Operation: "patch.convert", SourcePath: "foo.iso", OutputFormat: "bar.bps"}, errors.New("unsupported xdelta3 code table"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = SaveFailureReport(root, r); e != nil {
		t.Fatal(e)
	}
	if _, e = SaveFailureReport(root, r); e == nil {
		t.Fatal("report overwrite allowed")
	}
	list, e := RecentFailureReports(root, 5)
	if e != nil || len(list) != 1 || list[0].ID != r.ID {
		t.Fatalf("reports %v %v", list, e)
	}
	if r.ErrorCode != "unsupported_format" || r.PatchFormat != "" || r.TargetExtension != ".iso" || r.OutputFormat != ".bps" {
		t.Fatalf("unexpected report %#v", r)
	}
}
func TestFailureDoesNotRecordArbitraryMessages(t *testing.T) {
	secret := "key=123456789012345678901"
	r, e := NewFailureReport(FailureContext{Operation: "patch.preview"}, errors.New("bad "+secret))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(r.Summary(), secret) {
		t.Fatal("raw error leaked into summary")
	}
	if _, e := NewFailureReport(FailureContext{}, nil); e == nil {
		t.Fatal("should reject nil failure")
	}
}

func TestWarningReport(t *testing.T) {
	r, e := NewFailureReport(FailureContext{Operation: "pf.apply.warning", WarningCount: 3}, errors.New("skipped file D:\\private\\SECRET.cfg"))
	if e != nil {
		t.Fatal(e)
	}
	if r.Severity != "warning" || r.ErrorCode != "operation_warning" || r.WarningCount != 3 {
		t.Fatalf("invalid warning: %#v", r)
	}
	if strings.Contains(r.Summary(), "SECRET") {
		t.Fatal("warning included raw detail")
	}
}
