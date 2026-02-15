package service

import (
	"context"
	"sync"
	"time"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type State struct {
	ActiveGame domain.GameType `json:"active_game"`
	Phase      string          `json:"phase"` // "stopped", "running", "switching", ...
	UpdatedAt  time.Time       `json:"updated_at"`
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
			ActiveGame: "",
			Phase:      "stopped",
			UpdatedAt:  time.Now().UTC(),
		},
	}
}

func (m *memoryState) Get(ctx context.Context) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s, nil
}

func (m *memoryState) Set(ctx context.Context, s State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.UpdatedAt = time.Now().UTC()
	m.s = s
	return nil
}
