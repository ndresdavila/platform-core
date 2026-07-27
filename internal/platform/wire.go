package platform

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	httpapi "github.com/ndresdavila/platform-core/internal/adapters/http"
	"github.com/ndresdavila/platform-core/internal/adapters/postgres"
	"github.com/ndresdavila/platform-core/internal/application/identity"
	"github.com/ndresdavila/platform-core/internal/application/invoice"
	"github.com/ndresdavila/platform-core/internal/application/tenant"
)

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type UUIDGen struct{}

func (UUIDGen) New() uuid.UUID { return uuid.New() }

func NewHTTPServer(db *pgxpool.Pool) *httpapi.Server {
	tenants := postgres.NewTenantRepo(db)
	employees := postgres.NewEmployeeRepo(db)
	persons := postgres.NewPersonRepo(db)
	products := postgres.NewProductRepo(db)
	invoices := postgres.NewInvoiceRepo(db)
	clock := SystemClock{}
	ids := UUIDGen{}

	return &httpapi.Server{
		Tenants:  tenant.NewService(tenants, clock, ids),
		Identity: identity.NewService(tenants, employees, clock, ids),
		Invoices: invoice.NewService(tenants, persons, products, invoices, clock, ids),
	}
}
