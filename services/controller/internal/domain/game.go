package domain

import "context"

type GameType string

const (
	GameMinecraft GameType = "minecraft"
	GameHytale    GameType = "hytale"
)

type GameAdapter interface {
	Type() GameType
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Backup(ctx context.Context) (backupKey string, err error)
	Restore(ctx context.Context, backupKey string) error
	SendCommand(ctx context.Context, command string) error
	Status(ctx context.Context) (map[string]any, error)
}
