package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/vatsalpatel/hostbox/internal/api"
	"github.com/vatsalpatel/hostbox/internal/api/handlers"
	"github.com/vatsalpatel/hostbox/internal/api/routes"
	"github.com/vatsalpatel/hostbox/internal/config"
	"github.com/vatsalpatel/hostbox/internal/database"
	"github.com/vatsalpatel/hostbox/internal/logger"
	"github.com/vatsalpatel/hostbox/internal/repository"
	"github.com/vatsalpatel/hostbox/migrations"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Setup logger
	l := logger.Setup(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(l)

	// 3. Open database
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		l.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)

	// 4. Run migrations
	if err := database.Migrate(db, migrations.FS); err != nil {
		l.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 5. Initialize repositories
	repos := repository.New(db)

	// 6. Create and start server
	srv := api.NewServer(cfg, db, repos, l)

	// 7. Register routes
	healthHandler := handlers.NewHealthHandler(srv.StartTime(), db)
	routes.Register(srv.Echo, healthHandler)

	// 8. Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		l.Error("server error", "error", err)
		os.Exit(1)
	}
}
