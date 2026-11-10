package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Seen         []string `json:"seen"`
	SourceIndex  int      `json:"source_index"`
	mu           sync.Mutex
	seenSet      map[string]struct{}
	path         string
}

func loadState(path string) (*State, error) {
	s := &State{path: path, seenSet: map[string]struct{}{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	for _, id := range s.Seen {
		s.seenSet[id] = struct{}{}
	}
	return s, nil
}

func (s *State) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seenSet[id]
	return ok
}

func (s *State) Mark(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seenSet[id]; ok {
		return s.saveLocked()
	}
	s.seenSet[id] = struct{}{}
	s.Seen = append(s.Seen, id)
	return s.saveLocked()
}

func (s *State) AdvanceSource(n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 {
		return nil
	}
	s.SourceIndex = (s.SourceIndex + 1) % n
	return s.saveLocked()
}

func (s *State) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	data, err := json.MarshalIndent(struct {
		Seen        []string `json:"seen"`
		SourceIndex int      `json:"source_index"`
	}{s.Seen, s.SourceIndex}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
