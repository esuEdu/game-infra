package main

import (
	"log/slog"
	"os"

	"github.com/esuEdu/game-infra/controller/internal/adapters/hytale"
	"github.com/esuEdu/game-infra/controller/internal/adapters/minecraft"
	"github.com/esuEdu/game-infra/controller/internal/api"
	"github.com/esuEdu/game-infra/controller/internal/app"
	"github.com/esuEdu/game-infra/controller/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := app.LoadConfig()

	mc := minecraft.NewAdapter(log)
	hy := hytale.NewAdapter(log)

	controllerSvc := service.NewControllerService(
		log,
		service.NewMemoryState(),
		map[string]service.Adapter{
			"minecraft": mc,
			"hytale":    hy,
		},
	)

	a := app.New(log, cfg, controllerSvc)

	srv := api.NewServer(a)

	log.Info("http listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
	}
}
