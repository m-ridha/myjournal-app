package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestRunAndManualRename(t *testing.T) {
	var tag string
	patches := 0
	current := &drive.File{Id: "test-id", Version: 1, Md5Checksum: "a", ModifiedTime: "first", HeadRevisionId: "r1", MimeType: "text/plain"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Errorf("multipart upload: %v", err)
				w.WriteHeader(400)
				return
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			part, err := reader.NextPart()
			if err != nil {
				t.Errorf("metadata part: %v", err)
				w.WriteHeader(400)
				return
			}
			var f drive.File
			if err := json.NewDecoder(part).Decode(&f); err != nil {
				t.Errorf("metadata: %v", err)
			}
			tag = f.AppProperties["s4Run"]
			current.AppProperties = map[string]string{"s4Run": tag}
		case http.MethodPatch:
			patches++
			if patches == 1 {
				current.Version = 2
				current.Md5Checksum = "b"
				current.ModifiedTime = "second"
				current.HeadRevisionId = "r2"
			}
			if patches == 2 {
				current.Version = 3
			}
			if patches > 2 {
				t.Error("unexpected write: stale client must not upload")
			}
		case http.MethodGet:
			if !strings.Contains(r.URL.Query().Get("fields"), "md5Checksum") {
				t.Error("missing candidate fields")
			}
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("response: %v", err)
		}
	}))
	defer srv.Close()
	ds, err := drive.NewService(context.Background(), option.WithEndpoint(srv.URL+"/"), option.WithHTTPClient(srv.Client()), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	e := &experiment{baselinePath: filepath.Join(t.TempDir(), "baseline.json")}
	var out bytes.Buffer
	if err := e.run(context.Background(), ds, &out); err != nil {
		t.Fatalf("run: %v; %s", err, out.String())
	}
	for _, want := range []string{"stale_save_md5Checksum_rejected: true", "client_b_content_preserved: true", "candidate_md5Checksum_content_changed_rename_stable: true", "candidate_version_content_changed_rename_stable: false"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %s in %s", want, out.String())
		}
	}
	if patches != 2 || tag == "" {
		t.Fatalf("writes=%d, tagged=%t", patches, tag != "")
	}
	current.Version = 4 // Mock a manual metadata-only rename.
	out.Reset()
	if err := e.checkRename(context.Background(), ds, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "manual_rename_md5Checksum_changed: false") {
		t.Fatal(out.String())
	}
	if patches != 2 {
		t.Fatal("manual verification must be read-only")
	}
}

func TestMissingFieldsAndBaselineProtection(t *testing.T) {
	if stale("", "new") || stale("old", "") || stale("same", "same") {
		t.Fatal("missing or matching fields must not detect a conflict")
	}
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := saveBaseline(path, baseline{FileID: "original"}); err != nil {
		t.Fatal(err)
	}
	if err := saveBaseline(path, baseline{FileID: "replacement"}); err == nil {
		t.Fatal("must refuse to overwrite baseline")
	}
}
