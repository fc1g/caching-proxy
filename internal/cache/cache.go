package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirPerm = 0o700
)

type Store struct {
	dir string
}

type Entry struct {
	Body       []byte
	StatusCode int
	Header     map[string][]string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("create a directory named path: %w", err)
	}

	return &Store{
		dir: dir,
	}, nil
}

func (s *Store) Set(key string, e Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	file, err := os.CreateTemp(s.dir, "")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(file.Name())

	_, err = file.Write(data)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("write to the file: %w", err)
	}

	if err = file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(file.Name(), s.filePath(key)); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (s *Store) Get(key string) (Entry, error) {
	file, err := os.Open(s.filePath(key))
	if err != nil {
		return Entry{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var entry Entry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&entry); err != nil {
		return Entry{}, fmt.Errorf("decode cache entry: %w", err)
	}

	return entry, nil
}

func (s *Store) Clear() error {
	if err := os.RemoveAll(s.dir); err != nil {
		return fmt.Errorf("remove cache dir: %w", err)
	}

	if err := os.MkdirAll(s.dir, dirPerm); err != nil {
		return fmt.Errorf("create a directory named path: %w", err)
	}

	return nil
}

func GenerateCacheKey(method, url string) string {
	return fmt.Sprintf("%s %s", method, url)
}

func (s *Store) filePath(key string) string {
	keyBytes := sha256.Sum256([]byte(key))
	return filepath.Join(s.dir, hex.EncodeToString(keyBytes[:]))
}
