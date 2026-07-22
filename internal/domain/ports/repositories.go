package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	Create(ctx context.Context, t *domain.Tenant) error
}

type SettingsRepository interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error)
	Upsert(ctx context.Context, s *domain.TenantSettings) error
}

type CustomerRepository interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error)
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Customer, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Customer, error)
	Create(ctx context.Context, c *domain.Customer) error
	Update(ctx context.Context, c *domain.Customer) error
	UpsertByExternalID(ctx context.Context, c *domain.Customer) error
}

type EmployeeRepository interface {
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Employee, error)
	UpsertByExternalID(ctx context.Context, e *domain.Employee) error
}

type CatalogRepository interface {
	ListServices(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]domain.CatalogService, error)
	GetService(ctx context.Context, tenantID, id uuid.UUID) (*domain.CatalogService, error)
	CreateService(ctx context.Context, s *domain.CatalogService) error
}

type BookingRepository interface {
	Create(ctx context.Context, b *domain.Booking) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Booking, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Booking, error)
	ListByRange(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.Booking, error)
	Update(ctx context.Context, b *domain.Booking) error
}

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error)
	GetLatestByBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*domain.Payment, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Payment, error)
	Update(ctx context.Context, p *domain.Payment) error
}

type BankAccountRepository interface {
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]domain.BankAccount, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BankAccount, error)
	Create(ctx context.Context, a *domain.BankAccount) error
}

type DesignRepository interface {
	Create(ctx context.Context, d *domain.Design) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Design, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Design, error)
	Update(ctx context.Context, d *domain.Design) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int, error)
	CountByCustomerSince(ctx context.Context, tenantID, customerID uuid.UUID, since time.Time) (int, error)
}

type Clock interface {
	Now() time.Time
}

type IDGen interface {
	New() uuid.UUID
}
