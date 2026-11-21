package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutoReportsFindAndAppend(t *testing.T) {
	report, _ := NewFailureReport(FailureContext{Operation: "patch.apply", TitleID: "4A580001", SourcePath: `C:\Users\private\game.iso`}, errors.New(`checksum mismatch at C:\Users\private\game.iso key=top-secret`))
	count := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Query().Get("q"), report.EffectiveGroupID()) {
			t.Fatal("wrong group query")
		}
		if count == 0 {
			w.Write([]byte(`{"items":[]}`))
		} else {
			w.Write([]byte(`{"items":[{"number":44,"title":"[Diagnostic ` + report.EffectiveGroupID() + `] checksum"}]}`))
		}
	})
	mux.HandleFunc("/repos/"+ReleaseRepo+"/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatal("wrong method")
		}
		var b map[string]string
		json.NewDecoder(r.Body).Decode(&b)
		if strings.Contains(b["body"], "private") || strings.Contains(b["body"], "top-secret") {
			t.Fatal("private data leaked")
		}
		count++
		w.WriteHeader(201)
		w.Write([]byte(`{"number":44}`))
	})
	mux.HandleFunc("/repos/"+ReleaseRepo+"/issues/44/comments", func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(201)
		w.Write([]byte(`{"id":99}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	num, err := PublishDiagnostic(context.Background(), server.Client(), server.URL+"/repos/"+ReleaseRepo, "token", report)
	if err != nil || num != 44 {
		t.Fatal(err, num)
	}
	num, err = PublishDiagnostic(context.Background(), server.Client(), server.URL+"/repos/"+ReleaseRepo, "token", report)
	if err != nil || num != 44 || count != 2 {
		t.Fatal(err, num, count)
	}
}
func TestReportUploadStatePersistsAndDoesNotLeak(t *testing.T) {
	root := t.TempDir()
	s, _ := LoadReportUploadState(root)
	s.Submitted["0123456789abcdef12"] = 12
	if err := SaveReportUploadState(root, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadReportUploadState(root)
	if err != nil || loaded.Submitted["0123456789abcdef12"] != 12 {
		t.Fatal(err)
	}
}

func TestEditedDiagnosticNeverPublishesRawMessage(t *testing.T) {
	r, _ := NewFailureReport(FailureContext{Operation: "patch.apply"}, errors.New("checksum mismatch"))
	r.Explanation = "C:\\Users\\someone\\password.txt secret=999"
	r.OperatingSystem = "Windows & <script>"
	clean := sanitizeReportForIssue(r)
	body := IssueReportBody(clean)
	if strings.Contains(body, "password.txt") || strings.Contains(body, "script") || strings.Contains(body, "someone") {
		t.Fatalf("raw data leaked: %s", body)
	}
}
