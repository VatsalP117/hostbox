package handlers

import (
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/vatsalpatel/hostbox/internal/api/middleware"
	"github.com/vatsalpatel/hostbox/internal/dto"
	apperrors "github.com/vatsalpatel/hostbox/internal/errors"
	"github.com/vatsalpatel/hostbox/internal/models"
	"github.com/vatsalpatel/hostbox/internal/repository"
)

type DomainHandler struct {
	domainRepo   *repository.DomainRepository
	projectRepo  *repository.ProjectRepository
	activityRepo *repository.ActivityRepository
	config       struct{ PlatformDomain string }
	logger       *slog.Logger
}

func NewDomainHandler(
	domainRepo *repository.DomainRepository,
	projectRepo *repository.ProjectRepository,
	activityRepo *repository.ActivityRepository,
	platformDomain string,
	logger *slog.Logger,
) *DomainHandler {
	return &DomainHandler{
		domainRepo:   domainRepo,
		projectRepo:  projectRepo,
		activityRepo: activityRepo,
		config:       struct{ PlatformDomain string }{platformDomain},
		logger:       logger,
	}
}

func (h *DomainHandler) Create(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	var req dto.CreateDomainRequest
	if err := c.Bind(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := dto.ValidateStruct(&req); err != nil {
		return err
	}

	// Check domain uniqueness
	existing, err := h.domainRepo.GetByDomain(c.Request().Context(), req.Domain)
	if err == nil && existing != nil {
		return apperrors.NewConflict("Domain already registered")
	}

	domain := &models.Domain{
		ProjectID: project.ID,
		Domain:    req.Domain,
		Verified:  false,
	}

	if err := h.domainRepo.Create(c.Request().Context(), domain); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "domain.created",
		ResourceType: "domain",
		ResourceID:   &domain.ID,
	})

	return c.JSON(http.StatusCreated, dto.CreateDomainResponse{
		Domain: toDomainResponse(domain),
		DNSInstructions: dto.DNSInstructions{
			Type:  "CNAME",
			Name:  req.Domain,
			Value: h.config.PlatformDomain,
		},
	})
}

func (h *DomainHandler) ListByProject(c echo.Context) error {
	project, err := h.getOwnedProject(c, c.Param("projectId"))
	if err != nil {
		return err
	}

	domains, err := h.domainRepo.ListByProject(c.Request().Context(), project.ID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	data := make([]dto.DomainResponse, len(domains))
	for i, d := range domains {
		data[i] = toDomainResponse(&d)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"domains": data,
	})
}

func (h *DomainHandler) Delete(c echo.Context) error {
	domain, err := h.domainRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Domain")
		}
		return apperrors.NewInternal(err)
	}

	// Verify ownership via project
	_, err = h.getOwnedProject(c, domain.ProjectID)
	if err != nil {
		return err
	}

	if err := h.domainRepo.Delete(c.Request().Context(), domain.ID); err != nil {
		return apperrors.NewInternal(err)
	}

	user := middleware.GetUser(c)
	h.activityRepo.Create(c.Request().Context(), &models.ActivityLog{
		UserID:       &user.ID,
		Action:       "domain.deleted",
		ResourceType: "domain",
		ResourceID:   &domain.ID,
	})

	return c.JSON(http.StatusOK, dto.SuccessResponse{Success: true})
}

func (h *DomainHandler) Verify(c echo.Context) error {
	domain, err := h.domainRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return apperrors.NewNotFound("Domain")
		}
		return apperrors.NewInternal(err)
	}

	if _, err := h.getOwnedProject(c, domain.ProjectID); err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := h.domainRepo.UpdateLastChecked(c.Request().Context(), domain.ID, now); err != nil {
		return apperrors.NewInternal(err)
	}

	if _, err := net.LookupHost(domain.Domain); err != nil {
		return apperrors.NewBadRequest("Domain DNS record not found yet")
	}

	if err := h.domainRepo.UpdateVerification(c.Request().Context(), domain.ID, true, &now); err != nil {
		return apperrors.NewInternal(err)
	}

	domain.Verified = true
	domain.VerifiedAt = &now
	domain.LastCheckedAt = &now

	return c.JSON(http.StatusOK, map[string]interface{}{
		"domain": toDomainResponse(domain),
	})
}

func (h *DomainHandler) getOwnedProject(c echo.Context, projectID string) (*models.Project, error) {
	user := middleware.GetUser(c)
	project, err := h.projectRepo.GetByID(c.Request().Context(), projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NewNotFound("Project")
		}
		return nil, apperrors.NewInternal(err)
	}
	if project.OwnerID != user.ID && !user.IsAdmin {
		return nil, apperrors.NewForbidden("Access denied")
	}
	return project, nil
}

func toDomainResponse(d *models.Domain) dto.DomainResponse {
	resp := dto.DomainResponse{
		ID:        d.ID,
		ProjectID: d.ProjectID,
		Domain:    d.Domain,
		Verified:  d.Verified,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
	}
	if d.VerifiedAt != nil {
		s := d.VerifiedAt.Format(time.RFC3339)
		resp.VerifiedAt = &s
	}
	if d.LastCheckedAt != nil {
		s := d.LastCheckedAt.Format(time.RFC3339)
		resp.LastCheckedAt = &s
	}
	return resp
}
