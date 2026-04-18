package scheduler

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/VatsalP117/hostbox/internal/repository"
)

// Manager starts and coordinates all background schedulers.
type Manager struct {
	gc             *GarbageCollector
	sessionCleaner *SessionCleaner
	domainVerifier *DomainReVerifier
	logger         *slog.Logger
}

func NewManager(
	db *sql.DB,
	deployRepo *repository.DeploymentRepository,
	projectRepo *repository.ProjectRepository,
	settingsRepo *repository.SettingsRepository,
	domainRepo *repository.DomainRepository,
	logDir string,
	logger *slog.Logger,
) *Manager {
	return &Manager{
		gc:             NewGarbageCollector(deployRepo, projectRepo, settingsRepo, logDir, logger),
		sessionCleaner: NewSessionCleaner(db, logger),
		domainVerifier: NewDomainReVerifier(domainRepo, logger),
		logger:         logger,
	}
}

// Start launches all background schedulers as goroutines. Cancel the context to stop them.
func (m *Manager) Start(ctx context.Context) {
	go m.gc.Run(ctx)
	go m.sessionCleaner.Run(ctx)
	go m.domainVerifier.Run(ctx)
	m.logger.Info("background schedulers started",
		"gc_interval", "6h",
		"session_cleanup_interval", "1h",
		"domain_reverify_interval", "24h",
	)
}
