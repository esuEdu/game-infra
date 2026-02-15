package service

import (
	"context"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

func (c *ControllerService) switchWorkflow(ctx context.Context, from domain.GameType, to Adapter) (backupKey string, err error) {
	// stop current (if any)
	if from != "" {
		fromAd, err := c.adapterByType(from)
		if err != nil {
			return "", err
		}

		if err := fromAd.Stop(ctx); err != nil {
			return "", err
		}

		backupKey, err = fromAd.Backup(ctx)
		if err != nil {
			return "", err
		}
	}

	// restore target (optional: restore latest by game, etc.)

	// Here we do "no-op restore" unless you pass a key later.
	// If you want “restore latest”, you’d lookup latest key in S3/DDB here.
	// start target
	if err := to.Start(ctx); err != nil {
		return backupKey, err
	}

	return backupKey, nil
}
