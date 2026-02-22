package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Path     string    `json:"path"`
	OpenedAt time.Time `json:"openedAt"`
}

type Store struct {
	mu       sync.Mutex
	filePath string
	entries  []Entry
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		filePath: filepath.Join(dataDir, "history.json"),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.entries = []Entry{}
			return nil
		}
		return err
	}

	if len(data) == 0 {
		s.entries = []Entry{}
		return nil
	}

	return json.Unmarshal(data, &s.entries)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Record(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, Entry{
		Path:     filePath,
		OpenedAt: time.Now(),
	})

	return s.save()
}

func (s *Store) IsOpened(filePath string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.entries {
		if e.Path == filePath {
			return true
		}
	}
	return false
}

func (s *Store) GetLastFolder() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.entries) - 1; i >= 0; i-- {
		dir := filepath.Dir(s.entries[i].Path)
		return dir
	}
	return ""
}

func (s *Store) GetAll() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Entry, len(s.entries))
	copy(result, s.entries)
	return result
}
