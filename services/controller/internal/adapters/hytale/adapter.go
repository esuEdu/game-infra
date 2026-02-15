package hytale

import (
	"context"
	"log/slog"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type Adapter struct {
	log *slog.Logger
}

func NewAdapter(log *slog.Logger) *Adapter {
	return &Adapter{log: log}
}

func (a *Adapter) Type() domain.GameType { return domain.GameHytale }

func (a *Adapter) Start(ctx context.Context) error {
	a.log.Info("hytale start (stub)")
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.log.Info("hytale stop (stub)")
	return nil
}

func (a *Adapter) Backup(ctx context.Context) (string, error) {
	a.log.Info("hytale backup (stub)")
	return "s3://backups/hytale/latest.zip", nil
}

func (a *Adapter) Restore(ctx context.Context, backupKey string) error {
	a.log.Info("hytale restore (stub)", "backup", backupKey)
	return nil
}

func (a *Adapter) SendCommand(ctx context.Context, command string) error {
	a.log.Info("hytale command (stub)", "cmd", command)
	return nil
}

func (a *Adapter) Status(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"adapter": "hytale",
		"ready":   true,
		"note":    "stub until official tooling exists",
	}, nil
}
