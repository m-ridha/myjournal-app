package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// stagedFile is the minimal file I/O needed before committing an atomic save.
type stagedFile interface {
	io.WriteCloser
	Name() string
	Sync() error
}

func writeAtomically(ctx context.Context, path string, data []byte, rename func(string, string) error) error {
	return stageAtomicWrite(ctx, path, data, func(dir, pattern string) (stagedFile, error) {
		return os.CreateTemp(dir, pattern)
	}, rename)
}

// Stage in the destination directory so rename atomically replaces the entry.
// Readers see the complete old or new file; failures before rename preserve the old file.
func stageAtomicWrite(ctx context.Context, path string, data []byte, createTemp func(string, string) (stagedFile, error), rename func(string, string) error) error {
	temp, err := createTemp(filepath.Dir(path), ".entry-*")
	if err != nil {
		return err
	}
	defer func() { _ = temp.Close(); _ = os.Remove(temp.Name()) }()
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return rename(temp.Name(), path)
}
