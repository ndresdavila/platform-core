package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants ports.TenantRepository
	clock   ports.Clock
	ids     ports.IDGen
}

func NewService(tenants ports.TenantRepository, clock ports.Clock, ids ports.IDGen) *Service {
	return &Service{tenants: tenants, clock: clock, ids: ids}
}

type CreateInput struct {
	Slug string
	Name string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Tenant, error) {
	slug := strings.TrimSpace(strings.ToLower(in.Slug))
	name := strings.TrimSpace(in.Name)
	if slug == "" || name == "" {
		return nil, fmt.Errorf("%w: slug and name required", domain.ErrValidation)
	}
	t := &domain.Tenant{
		ID:        s.ids.New(),
		Slug:      slug,
		Name:      name,
		Active:    true,
		CreatedAt: s.clock.Now().UTC(),
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	if err := s.tenants.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return s.tenants.GetBySlug(ctx, strings.TrimSpace(strings.ToLower(slug)))
}
