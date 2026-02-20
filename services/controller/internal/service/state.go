package service

import (
	"context"
	"sync"
	"time"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type State struct {
	ActiveGame   domain.GameType   `json:"active_game"`
	Phase        string            `json:"phase"`
	LastBackups  map[string]string `json:"last_backups"`
	SourceByGame map[string]string `json:"source_by_game"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type StateStore interface {
	Get(ctx context.Context) (State, error)
	Set(ctx context.Context, s State) error
}

type memoryState struct {
	mu sync.Mutex
	s  State
}

func NewMemoryState() StateStore {
	return &memoryState{
		s: State{
			ActiveGame:   "",
			Phase:        "stopped",
			LastBackups:  map[string]string{},
			SourceByGame: map[string]string{},
			UpdatedAt:    time.Now().UTC(),
		},
	}
}

func (m *memoryState) Get(ctx context.Context) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneState(m.s), nil
}

func (m *memoryState) Set(ctx context.Context, s State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.UpdatedAt = time.Now().UTC()
	m.s = cloneState(s)
	return nil
}

func cloneState(s State) State {
	cp := s

	cp.LastBackups = map[string]string{}
	for k, v := range s.LastBackups {
		cp.LastBackups[k] = v
	}

	cp.SourceByGame = map[string]string{}
	for k, v := range s.SourceByGame {
		cp.SourceByGame[k] = v
	}

	return cp
}
