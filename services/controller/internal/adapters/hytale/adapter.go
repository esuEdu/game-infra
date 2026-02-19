package hytale

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type Adapter struct {
	log        *slog.Logger
	mu         sync.Mutex
	running    bool
	lastBackup string
	lastSource string
}

func NewAdapter(log *slog.Logger) *Adapter {
	return &Adapter{log: log}
}

func (a *Adapter) Type() domain.GameType { return domain.GameHytale }

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	a.log.Info("hytale start (stub)")
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	a.log.Info("hytale stop (stub)")
	return nil
}

func (a *Adapter) Backup(ctx context.Context) (string, error) {
	a.mu.Lock()
	a.lastBackup = "s3://backups/hytale/" + time.Now().UTC().Format("20060102-150405") + ".zip"
	backup := a.lastBackup
	a.mu.Unlock()
	a.log.Info("hytale backup (stub)", "backup", backup)
	return backup, nil
}

func (a *Adapter) Restore(ctx context.Context, backupKey string) error {
	a.mu.Lock()
	a.lastBackup = backupKey
	a.mu.Unlock()
	a.log.Info("hytale restore (stub)", "backup", backupKey)
	return nil
}

func (a *Adapter) SeedFromSource(ctx context.Context, sourceURL string) error {
	a.mu.Lock()
	a.lastSource = sourceURL
	a.mu.Unlock()
	a.log.Info("hytale seed from source (stub)", "source", sourceURL)
	return nil
}

func (a *Adapter) SyncToSource(ctx context.Context, sourceURL string) error {
	a.mu.Lock()
	a.lastSource = sourceURL
	a.mu.Unlock()
	a.log.Info("hytale sync to source (stub)", "source", sourceURL)
	return nil
}

func (a *Adapter) SendCommand(ctx context.Context, command string) error {
	a.log.Info("hytale command (stub)", "cmd", command)
	return nil
}

func (a *Adapter) Status(ctx context.Context) (map[string]any, error) {
	a.mu.Lock()
	running := a.running
	lastBackup := a.lastBackup
	lastSource := a.lastSource
	a.mu.Unlock()

	return map[string]any{
		"adapter":     "hytale",
		"ready":       true,
		"running":     running,
		"last_backup": lastBackup,
		"last_source": lastSource,
		"note":        "stub until official tooling exists",
	}, nil
}
