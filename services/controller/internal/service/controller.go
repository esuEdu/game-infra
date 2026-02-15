package service

import (
	"context"
	"log/slog"
	"sync"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type Adapter = domain.GameAdapter

type ControllerService struct {
	log      *slog.Logger
	state    StateStore
	adapters map[string]Adapter

	opMu sync.Mutex // prevents overlapping workflows
}

func NewControllerService(log *slog.Logger, state StateStore, adapters map[string]Adapter) *ControllerService {
	return &ControllerService{
		log:      log,
		state:    state,
		adapters: adapters,
	}
}

func (c *ControllerService) Start(ctx context.Context, game string) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	ad, ok := c.adapters[game]
	if !ok {
		return domain.ErrUnknownGameType
	}

	// If something is active, stop it first (simple policy)
	st, _ := c.state.Get(ctx)
	if st.ActiveGame != "" && st.ActiveGame != ad.Type() {
		if err := c.stopUnsafe(ctx, st.ActiveGame); err != nil {
			return err
		}
	}

	if err := ad.Start(ctx); err != nil {
		return err
	}

	_ = c.state.Set(ctx, State{ActiveGame: ad.Type(), Phase: "running"})
	return nil
}

func (c *ControllerService) Stop(ctx context.Context) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	st, _ := c.state.Get(ctx)
	if st.ActiveGame == "" {
		return domain.ErrNoActiveGame
	}

	if err := c.stopUnsafe(ctx, st.ActiveGame); err != nil {
		return err
	}
	_ = c.state.Set(ctx, State{ActiveGame: "", Phase: "stopped"})
	return nil
}

func (c *ControllerService) Switch(ctx context.Context, game string) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	target, ok := c.adapters[game]
	if !ok {
		return domain.ErrUnknownGameType
	}

	st, _ := c.state.Get(ctx)
	if st.ActiveGame == target.Type() {
		// already on target
		return nil
	}

	_ = c.state.Set(ctx, State{ActiveGame: st.ActiveGame, Phase: "switching"})

	backupKey, err := c.switchWorkflow(ctx, st.ActiveGame, target)
	if err != nil {
		_ = c.state.Set(ctx, State{ActiveGame: st.ActiveGame, Phase: "error"})
		return err
	}

	c.log.Info("switch complete", "from", st.ActiveGame, "to", target.Type(), "backup", backupKey)
	_ = c.state.Set(ctx, State{ActiveGame: target.Type(), Phase: "running"})
	return nil
}

func (c *ControllerService) Backup(ctx context.Context) (string, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	st, _ := c.state.Get(ctx)
	if st.ActiveGame == "" {
		return "", domain.ErrNoActiveGame
	}

	ad, err := c.adapterByType(st.ActiveGame)
	if err != nil {
		return "", err
	}
	return ad.Backup(ctx)
}

func (c *ControllerService) Command(ctx context.Context, cmd string) error {
	// commands can be frequent; still serialize to avoid weirdness
	c.opMu.Lock()
	defer c.opMu.Unlock()

	st, _ := c.state.Get(ctx)
	if st.ActiveGame == "" {
		return domain.ErrNoActiveGame
	}

	ad, err := c.adapterByType(st.ActiveGame)
	if err != nil {
		return err
	}
	return ad.SendCommand(ctx, cmd)
}

func (c *ControllerService) Status(ctx context.Context) (map[string]any, error) {
	st, _ := c.state.Get(ctx)

	out := map[string]any{
		"active_game": st.ActiveGame,
		"phase":       st.Phase,
		"updated_at":  st.UpdatedAt,
	}

	if st.ActiveGame != "" {
		ad, err := c.adapterByType(st.ActiveGame)
		if err == nil {
			adSt, err2 := ad.Status(ctx)
			if err2 == nil {
				out["game_status"] = adSt
			}
		}
	}

	return out, nil
}

func (c *ControllerService) stopUnsafe(ctx context.Context, t domain.GameType) error {
	ad, err := c.adapterByType(t)
	if err != nil {
		return err
	}
	return ad.Stop(ctx)
}

func (c *ControllerService) adapterByType(t domain.GameType) (Adapter, error) {
	for _, ad := range c.adapters {
		if ad.Type() == t {
			return ad, nil
		}
	}
	return nil, domain.ErrUnknownGameType
}
