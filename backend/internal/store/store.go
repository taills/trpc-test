package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/taills/trpc-test/backend/internal/engine"
)

type Store struct {
	mu   sync.RWMutex
	runs map[string]*engine.RunResult
	dir  string
}

func New(dir string) *Store {
	return &Store{
		runs: make(map[string]*engine.RunResult),
		dir:  dir,
	}
}

func (s *Store) Save(run *engine.RunResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.RunID] = run
	if s.dir != "" {
		data, err := json.MarshalIndent(run, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(s.dir, run.RunID+".json"), data, 0644)
	}
	return nil
}

func (s *Store) Get(runID string) (*engine.RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run %q not found", runID)
	}
	return run, nil
}

func (s *Store) List() []*engine.RunResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*engine.RunResult, 0, len(s.runs))
	for _, r := range s.runs {
		result = append(result, r)
	}
	return result
}
