package catalog

import (
	"context"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants ports.TenantRepository
	catalog ports.CatalogRepository
	ids     ports.IDGen
}

func NewService(tenants ports.TenantRepository, catalog ports.CatalogRepository, ids ports.IDGen) *Service {
	return &Service{tenants: tenants, catalog: catalog, ids: ids}
}

func (s *Service) ListServices(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]domain.CatalogService, error) {
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !t.Active {
		return nil, domain.ErrTenantInactive
	}
	return s.catalog.ListServices(ctx, tenantID, activeOnly)
}

type CreateServiceInput struct {
	TenantID    uuid.UUID
	Name        string
	Description string
	DurationMin int
	PriceCents  int
	Currency    string
}

func (s *Service) CreateService(ctx context.Context, in CreateServiceInput) (*domain.CatalogService, error) {
	t, err := s.tenants.GetByID(ctx, tenantID(in.TenantID))
	if err != nil {
		return nil, err
	}
	if !t.Active {
		return nil, domain.ErrTenantInactive
	}
	currency := in.Currency
	if currency == "" {
		currency = "USD"
	}
	svc := &domain.CatalogService{
		ID:          s.ids.New(),
		TenantID:    in.TenantID,
		Name:        in.Name,
		Description: in.Description,
		DurationMin: in.DurationMin,
		PriceCents:  in.PriceCents,
		Currency:    currency,
		Active:      true,
	}
	if err := s.catalog.CreateService(ctx, svc); err != nil {
		return nil, err
	}
	return svc, nil
}

func tenantID(id uuid.UUID) uuid.UUID { return id }
