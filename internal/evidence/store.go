package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ObjectStore interface {
	Put(context.Context, string, io.Reader, int64) (ObjectInfo, error)
	Delete(context.Context, string) error
}

type MemoryObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{objects: map[string][]byte{}}
}

func (s *MemoryObjectStore) Put(_ context.Context, key string, reader io.Reader, max int64) (ObjectInfo, error) {
	if max <= 0 {
		return ObjectInfo{}, ErrArtifactTooLarge
	}
	limited := &io.LimitedReader{R: reader, N: max + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return ObjectInfo{}, err
	}
	if int64(len(data)) > max {
		return ObjectInfo{}, ErrArtifactTooLarge
	}
	digest := sha256.Sum256(data)
	s.mu.Lock()
	s.objects[key] = bytes.Clone(data)
	s.mu.Unlock()
	return ObjectInfo{Key: key, SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}, nil
}

func (s *MemoryObjectStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.objects, key)
	s.mu.Unlock()
	return nil
}

type LocalObjectStore struct{ root string }

func NewLocalObjectStore(root string) (*LocalObjectStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("artifact root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, err
	}
	return &LocalObjectStore{root: absolute}, nil
}

func (s *LocalObjectStore) path(key string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(key, "/"))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid storage key")
	}
	path := filepath.Join(s.root, clean)
	relative, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	return path, nil
}

func (s *LocalObjectStore) Put(_ context.Context, key string, reader io.Reader, max int64) (ObjectInfo, error) {
	path, err := s.path(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return ObjectInfo{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return ObjectInfo{}, err
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
	}()
	hash := sha256.New()
	limited := &io.LimitedReader{R: reader, N: max + 1}
	written, err := io.Copy(io.MultiWriter(temporary, hash), limited)
	if err != nil {
		return ObjectInfo{}, err
	}
	if written > max {
		return ObjectInfo{}, ErrArtifactTooLarge
	}
	if err := temporary.Sync(); err != nil {
		return ObjectInfo{}, err
	}
	if err := temporary.Close(); err != nil {
		return ObjectInfo{}, err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, SizeBytes: written, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func (s *LocalObjectStore) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
