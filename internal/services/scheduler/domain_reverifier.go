package scheduler

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
)

const domainGracePeriodDays = 7

// DomainReVerifier periodically re-checks DNS for verified domains.
type DomainReVerifier struct {
	domainRepo *repository.DomainRepository
	logger     *slog.Logger
}

func NewDomainReVerifier(domainRepo *repository.DomainRepository, logger *slog.Logger) *DomainReVerifier {
	return &DomainReVerifier{domainRepo: domainRepo, logger: logger}
}

func (rv *DomainReVerifier) Run(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rv.logger.Info("domain re-verifier stopped")
			return
		case <-ticker.C:
			rv.ReVerify(ctx)
		}
	}
}

func (rv *DomainReVerifier) ReVerify(ctx context.Context) {
	domains, err := rv.domainRepo.ListVerified(ctx)
	if err != nil {
		rv.logger.Error("domain re-verification: failed to list", "error", err)
		return
	}
	rv.logger.Info("domain re-verification started", "count", len(domains))

	for _, d := range domains {
		rv.checkDomain(ctx, d)
	}
}

func (rv *DomainReVerifier) checkDomain(ctx context.Context, d models.Domain) {
	now := time.Now().UTC()
	_, err := net.LookupHost(d.Domain)

	if err != nil {
		rv.logger.Warn("domain DNS check failed",
			"domain", d.Domain,
			"project_id", d.ProjectID,
			"error", err,
		)

		if d.VerifiedAt != nil && time.Since(*d.VerifiedAt) > domainGracePeriodDays*24*time.Hour {
			if err := rv.domainRepo.UpdateVerification(ctx, d.ID, false, &now); err != nil {
				rv.logger.Error("failed to mark domain unverified", "domain", d.Domain, "error", err)
			} else {
				rv.logger.Warn("domain marked unverified (grace period expired)", "domain", d.Domain)
			}
		}
	} else {
		if err := rv.domainRepo.UpdateLastChecked(ctx, d.ID, now); err != nil {
			rv.logger.Error("failed to update last_checked_at", "domain", d.Domain, "error", err)
		}
	}
}
