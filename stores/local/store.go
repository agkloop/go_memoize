package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	memoize "github.com/agkloop/go_memoize"
)

// LocalFileStore[V] persists cache entries as Gob-encoded files in a directory.
// Each key maps to a file named by its SHA-256 hash (hex). Writes are atomic
// (write temp file → rename). Get returns stored entries for the cache engine
// to decide freshness and staleness.
//
// V must be Gob-encodable.
type LocalFileStore[V any] struct {
	dir string
}

// New creates a LocalFileStore that stores entries in dir.
func New[V any](dir string) *LocalFileStore[V] {
	return &LocalFileStore[V]{dir: dir}
}

type diskEntry[V any] struct {
	Stored memoize.Stored[V]
}

func (s *LocalFileStore[V]) path(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(s.dir, fmt.Sprintf("%x.cache", h))
}

func (s *LocalFileStore[V]) Get(_ context.Context, key string) (memoize.Stored[V], bool, error) {
	var zero memoize.Stored[V]
	data, err := os.ReadFile(s.path(key))
	if os.IsNotExist(err) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var de diskEntry[V]
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&de); err != nil {
		return zero, false, nil // treat corrupt file as miss
	}
	return de.Stored, true, nil
}

func (s *LocalFileStore[V]) Set(_ context.Context, key string, value memoize.Stored[V]) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(diskEntry[V]{Stored: value}); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".tmp-")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path(key))
}

func (s *LocalFileStore[V]) Delete(_ context.Context, key string) error {
	err := os.Remove(s.path(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalFileStore[V]) Clear(_ context.Context) error {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(s.dir, e.Name()))
	}
	return nil
}
