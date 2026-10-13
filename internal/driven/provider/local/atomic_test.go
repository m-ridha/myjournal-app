package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failingStagedFile struct {
	*os.File
	stage   string
	failure error
}

var _ stagedFile = (*failingStagedFile)(nil)

func (f *failingStagedFile) Write(data []byte) (int, error) {
	if f.stage == "write" {
		return 0, f.failure
	}
	return f.File.Write(data)
}
func (f *failingStagedFile) Sync() error {
	if f.stage == "sync" {
		return f.failure
	}
	return f.File.Sync()
}
func (f *failingStagedFile) Close() error {
	err := f.File.Close()
	if f.stage == "close" {
		return f.failure
	}
	return err
}

func TestAtomicStagingFailuresPreserveOldTextAndRemoveTemporaryFiles(t *testing.T) {
	for _, stage := range []string{"write", "sync", "close"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "entry.txt")
			if err := os.WriteFile(path, []byte("old synthetic text"), 0600); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("synthetic staging failure")
			create := func(dir, pattern string) (stagedFile, error) {
				file, err := os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				return &failingStagedFile{File: file, stage: stage, failure: failure}, nil
			}
			committed := false
			rename := func(from, to string) error { committed = true; return os.Rename(from, to) }
			if err := stageAtomicWrite(context.Background(), path, []byte("new synthetic text"), create, rename); !errors.Is(err, failure) {
				t.Fatal("staging error identity lost")
			}
			if committed {
				t.Fatal("failed staging committed replacement")
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "old synthetic text" {
				t.Fatal("old text lost after staging failure")
			}
			if entries, err := os.ReadDir(root); err != nil || len(entries) != 1 || entries[0].Name() != "entry.txt" {
				t.Fatal("staging failure leaked temporary file")
			}
		})
	}
}
