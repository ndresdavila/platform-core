package design

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants  ports.TenantRepository
	designs  ports.DesignRepository
	settings ports.SettingsRepository
	clock    ports.Clock
	ids      ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	designs ports.DesignRepository,
	settings ports.SettingsRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{tenants: tenants, designs: designs, settings: settings, clock: clock, ids: ids}
}

type CreateInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Prompt     string
	PhotoURL   string
	ModelUsed  string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Design, error) {
	st, err := s.settings.Get(ctx, in.TenantID)
	if err != nil {
		return nil, err
	}
	if !st.FeatureDesignAI {
		return nil, fmt.Errorf("%w: design AI disabled", domain.ErrForbidden)
	}
	total, err := s.designs.CountByCustomer(ctx, in.TenantID, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if total >= st.DesignAIMaxTotal {
		return nil, fmt.Errorf("%w: max designs reached", domain.ErrValidation)
	}
	// America/Guayaquil approx: UTC-5
	now := s.clock.Now()
	loc := time.FixedZone("ECT", -5*3600)
	startDay := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, loc).UTC()
	daily, err := s.designs.CountByCustomerSince(ctx, in.TenantID, in.CustomerID, startDay)
	if err != nil {
		return nil, err
	}
	if daily >= st.DesignAIDailyLimit {
		return nil, fmt.Errorf("%w: daily design limit", domain.ErrValidation)
	}
	model := in.ModelUsed
	if model == "" {
		model = "mock"
	}
	d := &domain.Design{
		ID: s.ids.New(), TenantID: in.TenantID, CustomerID: in.CustomerID,
		Prompt: in.Prompt, PhotoURL: in.PhotoURL, ModelUsed: model,
		Status: domain.DesignProcessing, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.designs.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Complete(ctx context.Context, tenantID, id uuid.UUID, resultURL string, failed bool, errMsg string) (*domain.Design, error) {
	d, err := s.designs.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	d.UpdatedAt = now
	d.ProcessedAt = &now
	if failed {
		d.Status = domain.DesignFailed
		d.ErrorMessage = errMsg
	} else {
		d.Status = domain.DesignReady
		d.ResultURL = resultURL
	}
	if err := s.designs.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) List(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Design, error) {
	return s.designs.ListByCustomer(ctx, tenantID, customerID)
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Design, error) {
	return s.designs.GetByID(ctx, tenantID, id)
}

func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.designs.Delete(ctx, tenantID, id)
}

func (s *Service) Quota(ctx context.Context, tenantID, customerID uuid.UUID) (used, limit, totalUsed, totalMax int, err error) {
	st, err := s.settings.Get(ctx, tenantID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	now := s.clock.Now()
	loc := time.FixedZone("ECT", -5*3600)
	startDay := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, loc).UTC()
	used, err = s.designs.CountByCustomerSince(ctx, tenantID, customerID, startDay)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	totalUsed, err = s.designs.CountByCustomer(ctx, tenantID, customerID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return used, st.DesignAIDailyLimit, totalUsed, st.DesignAIMaxTotal, nil
}
