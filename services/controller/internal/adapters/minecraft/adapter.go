package minecraft

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

func (a *Adapter) Type() domain.GameType { return domain.GameMinecraft }

func (a *Adapter) Start(ctx context.Context) error {
	a.log.Info("minecraft start (stub)")
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.log.Info("minecraft stop (stub)")
	return nil
}

func (a *Adapter) Backup(ctx context.Context) (string, error) {
	a.log.Info("minecraft backup (stub)")
	return "s3://backups/minecraft/latest.zip", nil
}

func (a *Adapter) Restore(ctx context.Context, backupKey string) error {
	a.log.Info("minecraft restore (stub)", "backup", backupKey)
	return nil
}

func (a *Adapter) SendCommand(ctx context.Context, command string) error {
	a.log.Info("minecraft command (stub)", "cmd", command)
	return nil
}

func (a *Adapter) Status(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"adapter": "minecraft",
		"ready":   true,
	}, nil
}
