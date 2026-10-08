package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const metadataFields = "id,version,modifiedTime,md5Checksum,headRevisionId,appProperties,trashed,mimeType"

type baseline struct {
	FileID string
	RunID  string
	Values map[string]string
}

func values(f *drive.File) map[string]string {
	v := ""
	if f.Version != 0 {
		v = strconv.FormatInt(f.Version, 10)
	}
	return map[string]string{"version": v, "modifiedTime": f.ModifiedTime, "md5Checksum": f.Md5Checksum, "headRevisionId": f.HeadRevisionId}
}

var candidates = []string{"version", "modifiedTime", "md5Checksum", "headRevisionId"}

func report(w io.Writer, stage string, before, after map[string]string) {
	for _, field := range candidates {
		present := before[field] != "" && after[field] != ""
		_, _ = fmt.Fprintf(w, "%s_%s_available: %t\n%s_%s_changed: %t\n", stage, field, present, stage, field, present && before[field] != after[field])
	}
}

func get(ctx context.Context, ds *drive.Service, id string) (*drive.File, error) {
	return ds.Files.Get(id).Fields(metadataFields).Context(ctx).Do()
}

// A missing field never qualifies as a usable conflict token.
func stale(base, current string) bool { return base != "" && current != "" && base != current }

func stageError(w io.Writer, stage string, err error) error {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		_, _ = fmt.Fprintf(w, "failed_stage: %s\ngoogle_http_status: %d\n", stage, apiErr.Code)
	} else {
		_, _ = fmt.Fprintf(w, "failed_stage: %s\n", stage)
	}
	return fmt.Errorf("%s failed", stage)
}

func (e *experiment) run(ctx context.Context, ds *drive.Service, w io.Writer) error {
	runID, err := newRunID()
	if err != nil {
		return stageError(w, "run_id", err)
	}
	name := "myjournal-s4-" + runID + ".txt"
	f, err := ds.Files.Create(&drive.File{Name: name, MimeType: "text/plain", AppProperties: map[string]string{"s4Run": runID}}).
		Media(strings.NewReader("S4 synthetic content A\n")).Fields(metadataFields).Context(ctx).Do()
	if err != nil {
		return stageError(w, "create", err)
	}
	// Show the identifier immediately so a partial run remains inspectable.
	_, _ = fmt.Fprintf(w, "test_file_id: %s\n", f.Id)
	a, err := get(ctx, ds, f.Id)
	if err != nil {
		return stageError(w, "client_a_load", err)
	}
	_, err = ds.Files.Update(f.Id, &drive.File{}).Media(strings.NewReader("S4 synthetic content B\n")).Fields("id").Context(ctx).Do()
	if err != nil {
		return stageError(w, "client_b_update", err)
	}
	start := time.Now()
	b, err := get(ctx, ds, f.Id)
	elapsed := time.Since(start)
	if err != nil {
		return stageError(w, "client_a_pre_save_check", err)
	}
	av, bv := values(a), values(b)
	report(w, "content_edit", av, bv)
	_, _ = fmt.Fprintf(w, "metadata_check_requests: 1\nmetadata_check_ms: %d\n", elapsed.Milliseconds())
	for _, field := range candidates {
		_, _ = fmt.Fprintf(w, "stale_save_%s_rejected: %t\n", field, stale(av[field], bv[field]))
	}
	// A's stale content is never uploaded. Re-read to confirm B remains current.
	current, err := get(ctx, ds, f.Id)
	if err != nil {
		return stageError(w, "verify_no_stale_write", err)
	}
	_, _ = fmt.Fprintf(w, "client_b_content_preserved: %t\n", bv["md5Checksum"] != "" && current.Md5Checksum == bv["md5Checksum"])
	_, err = ds.Files.Update(f.Id, &drive.File{Name: "myjournal-s4-" + runID + "-renamed.txt"}).Fields("id").Context(ctx).Do()
	if err != nil {
		return stageError(w, "api_rename", err)
	}
	renamed, err := get(ctx, ds, f.Id)
	if err != nil {
		return stageError(w, "read_after_api_rename", err)
	}
	rv := values(renamed)
	report(w, "api_rename", bv, rv)
	for _, field := range candidates {
		eligible := stale(av[field], bv[field]) && bv[field] != "" && rv[field] == bv[field]
		_, _ = fmt.Fprintf(w, "candidate_%s_content_changed_rename_stable: %t\n", field, eligible)
	}
	if err := saveBaseline(e.baselinePath, baseline{FileID: f.Id, RunID: runID, Values: rv}); err != nil {
		return stageError(w, "save_local_baseline", err)
	}
	_, _ = fmt.Fprintln(w, "S4_API_EXPERIMENT: completed; perform the manual Drive rename check next")
	return nil
}

func saveBaseline(path string, b baseline) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	encodeErr := json.NewEncoder(f).Encode(b)
	return errors.Join(encodeErr, f.Close())
}

func (e *experiment) checkRename(ctx context.Context, ds *drive.Service, w io.Writer) error {
	f, err := os.Open(e.baselinePath)
	if err != nil {
		return stageError(w, "open_baseline", err)
	}
	var b baseline
	err = json.NewDecoder(io.LimitReader(f, 8192)).Decode(&b)
	err = errors.Join(err, f.Close())
	if err != nil || b.FileID == "" || b.RunID == "" || len(b.Values) != 4 {
		return stageError(w, "validate_baseline", fmt.Errorf("invalid metadata baseline"))
	}
	current, err := get(ctx, ds, b.FileID)
	if err != nil {
		return stageError(w, "read_after_manual_rename", err)
	}
	if current.Trashed || current.MimeType != "text/plain" || current.AppProperties["s4Run"] != b.RunID {
		return stageError(w, "verify_test_file_identity", fmt.Errorf("test file identity mismatch"))
	}
	report(w, "manual_rename", b.Values, values(current))
	_, _ = fmt.Fprintln(w, "S4_MANUAL_RENAME: observed; confirm that a rename was performed before interpreting these flags")
	return nil
}
