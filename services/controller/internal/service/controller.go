package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type Adapter = domain.GameAdapter

type latestBackupProvider interface {
	LatestBackup(ctx context.Context) (string, error)
}

type StartResult struct {
	Started string `json:"started"`
	Source  string `json:"source"` // data_url | backup
	Backup  string `json:"backup,omitempty"`
	DataURL string `json:"data_url,omitempty"`
}

type StopResult struct {
	Stopped bool   `json:"stopped"`
	Backup  string `json:"backup"`
	Synced  bool   `json:"synced"`
	DataURL string `json:"data_url,omitempty"`
}

type ControllerService struct {
	log      *slog.Logger
	state    StateStore
	adapters map[string]Adapter

	opMu sync.Mutex
}

func NewControllerService(log *slog.Logger, state StateStore, adapters map[string]Adapter) *ControllerService {
	return &ControllerService{
		log:      log,
		state:    state,
		adapters: adapters,
	}
}

func (c *ControllerService) Start(ctx context.Context, game string, dataURL string) (StartResult, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	ad, ok := c.adapters[game]
	if !ok {
		return StartResult{}, domain.ErrUnknownGameType
	}

	st, _ := c.state.Get(ctx)
	st = ensureStateMaps(st)

	// If another game is active, stop it, backup it, and sync to existing source.
	if st.ActiveGame != "" && st.ActiveGame != ad.Type() {
		previous, err := c.adapterByType(st.ActiveGame)
		if err != nil {
			return StartResult{}, err
		}
		if err := previous.Stop(ctx); err != nil {
			return StartResult{}, err
		}
		backupKey, err := previous.Backup(ctx)
		if err != nil {
			return StartResult{}, err
		}
		st.LastBackups[string(st.ActiveGame)] = backupKey

		if sourceURL := st.SourceByGame[string(st.ActiveGame)]; sourceURL != "" {
			if err := previous.SyncToSource(ctx, sourceURL); err != nil {
				return StartResult{}, err
			}
		}
	}

	result := StartResult{
		Started: game,
	}

	dataURL = strings.TrimSpace(dataURL)
	if dataURL != "" {
		if err := ad.SeedFromSource(ctx, dataURL); err != nil {
			return StartResult{}, err
		}
		st.SourceByGame[game] = dataURL
		result.Source = "data_url"
		result.DataURL = dataURL
	} else {
		backupKey, ok := st.LastBackups[game]
		if !ok || strings.TrimSpace(backupKey) == "" {
			provider, hasProvider := ad.(latestBackupProvider)
			if !hasProvider {
				return StartResult{}, domain.ErrNoBackupForGame
			}
			var err error
			backupKey, err = provider.LatestBackup(ctx)
			if err != nil || strings.TrimSpace(backupKey) == "" {
				return StartResult{}, domain.ErrNoBackupForGame
			}
			st.LastBackups[game] = backupKey
		}
		if err := ad.Restore(ctx, backupKey); err != nil {
			return StartResult{}, err
		}
		result.Source = "backup"
		result.Backup = backupKey
	}

	if err := ad.Start(ctx); err != nil {
		return StartResult{}, err
	}

	st.ActiveGame = ad.Type()
	st.Phase = "running"
	_ = c.state.Set(ctx, st)
	return result, nil
}

func (c *ControllerService) Stop(ctx context.Context) (StopResult, error) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	st, _ := c.state.Get(ctx)
	st = ensureStateMaps(st)
	if st.ActiveGame == "" {
		return StopResult{}, domain.ErrNoActiveGame
	}

	ad, err := c.adapterByType(st.ActiveGame)
	if err != nil {
		return StopResult{}, err
	}

	if err := ad.Stop(ctx); err != nil {
		return StopResult{}, err
	}

	backupKey, err := ad.Backup(ctx)
	if err != nil {
		return StopResult{}, err
	}

	gameKey := string(st.ActiveGame)
	st.LastBackups[gameKey] = backupKey

	result := StopResult{
		Stopped: true,
		Backup:  backupKey,
		Synced:  false,
	}

	if sourceURL := st.SourceByGame[gameKey]; sourceURL != "" {
		if err := ad.SyncToSource(ctx, sourceURL); err != nil {
			return StopResult{}, err
		}
		result.Synced = true
		result.DataURL = sourceURL
	}

	st.ActiveGame = ""
	st.Phase = "stopped"
	_ = c.state.Set(ctx, st)
	return result, nil
}

func (c *ControllerService) Switch(ctx context.Context, game string) error {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	target, ok := c.adapters[game]
	if !ok {
		return domain.ErrUnknownGameType
	}

	st, _ := c.state.Get(ctx)
	st = ensureStateMaps(st)
	if st.ActiveGame == target.Type() {
		return nil
	}

	st.Phase = "switching"
	_ = c.state.Set(ctx, st)

	backupKey, err := c.switchWorkflow(ctx, st.ActiveGame, target)
	if err != nil {
		st.Phase = "error"
		_ = c.state.Set(ctx, st)
		return err
	}

	if st.ActiveGame != "" && strings.TrimSpace(backupKey) != "" {
		st.LastBackups[string(st.ActiveGame)] = backupKey
	}

	c.log.Info("switch complete", "from", st.ActiveGame, "to", target.Type(), "backup", backupKey)
	st.ActiveGame = target.Type()
	st.Phase = "running"
	_ = c.state.Set(ctx, st)
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
	st = ensureStateMaps(st)

	out := map[string]any{
		"active_game":    st.ActiveGame,
		"phase":          st.Phase,
		"last_backups":   st.LastBackups,
		"source_by_game": st.SourceByGame,
		"updated_at":     st.UpdatedAt,
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

func (c *ControllerService) adapterByType(t domain.GameType) (Adapter, error) {
	for _, ad := range c.adapters {
		if ad.Type() == t {
			return ad, nil
		}
	}
	return nil, domain.ErrUnknownGameType
}

func ensureStateMaps(st State) State {
	if st.LastBackups == nil {
		st.LastBackups = map[string]string{}
	}
	if st.SourceByGame == nil {
		st.SourceByGame = map[string]string{}
	}
	return st
}
