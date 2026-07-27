package identity

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants   ports.TenantRepository
	employees ports.EmployeeRepository
	clock     ports.Clock
	ids       ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	employees ports.EmployeeRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{tenants: tenants, employees: employees, clock: clock, ids: ids}
}

type UpsertEmployeeInput struct {
	TenantID   uuid.UUID
	ExternalID string
	Email      string
	FirstName  string
	LastName   string
	Role       string
}

func (s *Service) UpsertEmployee(ctx context.Context, in UpsertEmployeeInput) (*domain.Employee, error) {
	if _, err := s.tenants.GetByID(ctx, in.TenantID); err != nil {
		return nil, err
	}
	role := in.Role
	if role == "" {
		role = "RECEPCIONISTA"
	}
	now := s.clock.Now()
	e := &domain.Employee{
		ID: s.ids.New(), TenantID: in.TenantID, ExternalID: in.ExternalID,
		Email: strings.ToLower(in.Email), FirstName: in.FirstName, LastName: in.LastName,
		Role: role, Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.employees.UpsertByExternalID(ctx, e); err != nil {
		return nil, err
	}
	return s.employees.GetByExternalID(ctx, in.TenantID, in.ExternalID)
}
