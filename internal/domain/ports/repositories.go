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

type EmployeeRepository interface {
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Employee, error)
	UpsertByExternalID(ctx context.Context, e *domain.Employee) error
}

type PersonRepository interface {
	Create(ctx context.Context, p *domain.Person) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Person, error)
	List(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Person, error)
}

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Product, error)
}

type InvoiceRepository interface {
	NextDocumentNumber(ctx context.Context, tenantID uuid.UUID, establishment, emissionPoint, docType string) (string, error)
	Create(ctx context.Context, doc *domain.ElectronicDocument) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ElectronicDocument, error)
	List(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ElectronicDocument, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ElectronicDocumentStatus, sriMessage, accessKey string) error
}

type Clock interface {
	Now() time.Time
}

type IDGen interface {
	New() uuid.UUID
}
