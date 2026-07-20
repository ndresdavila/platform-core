package platform

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	httpapi "github.com/ndresdavila/platform-core/internal/adapters/http"
	"github.com/ndresdavila/platform-core/internal/adapters/postgres"
	"github.com/ndresdavila/platform-core/internal/application/booking"
	"github.com/ndresdavila/platform-core/internal/application/catalog"
	"github.com/ndresdavila/platform-core/internal/application/design"
	"github.com/ndresdavila/platform-core/internal/application/identity"
	"github.com/ndresdavila/platform-core/internal/application/payment"
	"github.com/ndresdavila/platform-core/internal/application/tenant"
)

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type UUIDGen struct{}

func (UUIDGen) New() uuid.UUID { return uuid.New() }

func NewHTTPServer(db *pgxpool.Pool) *httpapi.Server {
	tenants := postgres.NewTenantRepo(db)
	customers := postgres.NewCustomerRepo(db)
	employees := postgres.NewEmployeeRepo(db)
	settings := postgres.NewSettingsRepo(db)
	catalogRepo := postgres.NewCatalogRepo(db)
	bookings := postgres.NewBookingRepo(db)
	payments := postgres.NewPaymentRepo(db)
	banks := postgres.NewBankAccountRepo(db)
	designs := postgres.NewDesignRepo(db)
	clock := SystemClock{}
	ids := UUIDGen{}

	return &httpapi.Server{
		Tenants:  tenant.NewService(tenants, clock, ids),
		Identity: identity.NewService(tenants, customers, employees, settings, clock, ids),
		Bookings: booking.NewService(tenants, customers, catalogRepo, bookings, payments, clock, ids),
		Payments: payment.NewService(tenants, payments, bookings, banks, clock, ids),
		Catalog:  catalog.NewService(tenants, catalogRepo, ids),
		Designs:  design.NewService(tenants, designs, settings, clock, ids),
	}
}
