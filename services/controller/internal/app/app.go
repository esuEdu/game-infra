package app

import (
	"log/slog"

	"github.com/esuEdu/game-infra/controller/internal/service"
)

type App struct {
	Log        *slog.Logger
	Config     Config
	Controller *service.ControllerService
}

func New(log *slog.Logger, cfg Config, controller *service.ControllerService) *App {
	return &App{
		Log:        log,
		Config:     cfg,
		Controller: controller,
	}
}
