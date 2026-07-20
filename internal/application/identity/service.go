package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants   ports.TenantRepository
	customers ports.CustomerRepository
	employees ports.EmployeeRepository
	settings  ports.SettingsRepository
	clock     ports.Clock
	ids       ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	customers ports.CustomerRepository,
	employees ports.EmployeeRepository,
	settings ports.SettingsRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{tenants: tenants, customers: customers, employees: employees, settings: settings, clock: clock, ids: ids}
}

type UpsertCustomerInput struct {
	TenantID   uuid.UUID
	ExternalID string
	Email      string
	FirstName  string
	LastName   string
	Phone      string
	NationalID *string
	Role       string
}

func (s *Service) UpsertCustomer(ctx context.Context, in UpsertCustomerInput) (*domain.Customer, error) {
	if _, err := s.tenants.GetByID(ctx, in.TenantID); err != nil {
		return nil, err
	}
	role := in.Role
	if role == "" {
		role = "CLIENTE"
	}
	now := s.clock.Now()
	name := strings.TrimSpace(in.FirstName + " " + in.LastName)
	c := &domain.Customer{
		ID:         s.ids.New(),
		TenantID:   in.TenantID,
		ExternalID: in.ExternalID,
		NationalID: in.NationalID,
		Email:      strings.ToLower(strings.TrimSpace(in.Email)),
		Name:       name,
		FirstName:  in.FirstName,
		LastName:   in.LastName,
		Phone:      in.Phone,
		Role:       role,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.customers.UpsertByExternalID(ctx, c); err != nil {
		return nil, err
	}
	return s.customers.GetByID(ctx, in.TenantID, c.ID)
}

func (s *Service) GetCustomer(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	return s.customers.GetByID(ctx, tenantID, id)
}

func (s *Service) GetCustomerByExternal(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Customer, error) {
	return s.customers.GetByExternalID(ctx, tenantID, externalID)
}

func (s *Service) UpdateCustomerProfile(ctx context.Context, tenantID, id uuid.UUID, nationalID, firstName, lastName, phone string) (*domain.Customer, error) {
	c, err := s.customers.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if nationalID == "" {
		return nil, fmt.Errorf("%w: national_id required", domain.ErrValidation)
	}
	nid := nationalID
	c.NationalID = &nid
	if firstName != "" {
		c.FirstName = firstName
	}
	if lastName != "" {
		c.LastName = lastName
	}
	if phone != "" {
		c.Phone = phone
	}
	c.Name = strings.TrimSpace(c.FirstName + " " + c.LastName)
	c.UpdatedAt = s.clock.Now()
	if err := s.customers.Update(ctx, c); err != nil {
		return nil, err
	}
	return s.customers.GetByID(ctx, tenantID, id)
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

func (s *Service) Settings(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error) {
	return s.settings.Get(ctx, tenantID)
}

func (s *Service) UpsertSettings(ctx context.Context, st *domain.TenantSettings) error {
	return s.settings.Upsert(ctx, st)
}
