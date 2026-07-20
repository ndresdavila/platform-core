package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/ndresdavila/platform-core/internal/adapters/postgres"
	"github.com/ndresdavila/platform-core/internal/application/catalog"
	"github.com/ndresdavila/platform-core/internal/application/payment"
	"github.com/ndresdavila/platform-core/internal/application/tenant"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/platform"
)

func main() {
	ctx := context.Background()
	db, err := postgres.NewPool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	featureDesignAI, err := featureFlag("FEATURE_DISENO_IA")
	if err != nil {
		log.Fatal(err)
	}

	clock := platform.SystemClock{}
	ids := platform.UUIDGen{}
	tenantRepo := postgres.NewTenantRepo(db)
	tenantService := tenant.NewService(tenantRepo, clock, ids)

	t, err := tenantService.GetBySlug(ctx, "nails-demo")
	if errors.Is(err, domain.ErrNotFound) {
		t, err = tenantService.Create(ctx, tenant.CreateInput{Slug: "nails-demo", Name: "Nails Demo"})
	}
	if err != nil {
		log.Fatalf("ensure tenant: %v", err)
	}

	settings := postgres.NewSettingsRepo(db)
	if err := settings.Upsert(ctx, &domain.TenantSettings{
		TenantID: t.ID, FeatureDesignAI: featureDesignAI, DesignAIDailyLimit: 3, DesignAIMaxTotal: 5,
	}); err != nil {
		log.Fatalf("seed settings: %v", err)
	}

	catalogService := catalog.NewService(tenantRepo, postgres.NewCatalogRepo(db), ids)
	for _, service := range []catalog.CreateServiceInput{
		{Name: "Manicure clásico", Description: "Limpieza, limado y esmaltado tradicional", DurationMin: 45, PriceCents: 1200, Currency: "USD"},
		{Name: "Manicure gel", Description: "Esmaltado en gel de larga duración", DurationMin: 60, PriceCents: 1800, Currency: "USD"},
		{Name: "Uñas acrílicas", Description: "Extensión y diseño personalizado", DurationMin: 90, PriceCents: 2800, Currency: "USD"},
		{Name: "Pedicure spa", Description: "Cuidado completo de pies", DurationMin: 60, PriceCents: 2000, Currency: "USD"},
	} {
		var exists bool
		if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM catalog_services WHERE tenant_id=$1 AND name=$2)`, t.ID, service.Name).Scan(&exists); err != nil {
			log.Fatalf("check service %q: %v", service.Name, err)
		}
		if !exists {
			service.TenantID = t.ID
			if _, err := catalogService.CreateService(ctx, service); err != nil {
				log.Fatalf("seed service %q: %v", service.Name, err)
			}
		}
	}

	paymentService := payment.NewService(tenantRepo, postgres.NewPaymentRepo(db), postgres.NewBookingRepo(db), postgres.NewBankAccountRepo(db), clock, ids)
	for _, account := range []domain.BankAccount{
		{BankCode: "PRODUBANCO", HolderName: "Uñas Prototipo Salón S.A.", AccountNumber: "1203456789", AccountType: "ahorros", TaxID: "0999999999001", NotifyEmail: "pagos@unasprototipo.ec", Active: true, SortOrder: 1},
		{BankCode: "BANCO_DEL_PACIFICO", HolderName: "Uñas Prototipo Salón S.A.", AccountNumber: "7654321098", AccountType: "corriente", TaxID: "0999999999001", NotifyEmail: "pagos@unasprototipo.ec", Active: true, SortOrder: 2},
		{BankCode: "BANCO_DE_GUAYAQUIL", HolderName: "Uñas Prototipo Salón S.A.", AccountNumber: "0034567890", AccountType: "ahorros", TaxID: "0999999999001", NotifyEmail: "pagos@unasprototipo.ec", Active: true, SortOrder: 3},
	} {
		var exists bool
		if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bank_accounts WHERE tenant_id=$1 AND bank_code=$2 AND account_number=$3)`, t.ID, account.BankCode, account.AccountNumber).Scan(&exists); err != nil {
			log.Fatalf("check bank account %s/%s: %v", account.BankCode, account.AccountNumber, err)
		}
		if !exists {
			account.TenantID = t.ID
			if err := paymentService.CreateBank(ctx, &account); err != nil {
				log.Fatalf("seed bank account %s/%s: %v", account.BankCode, account.AccountNumber, err)
			}
		}
	}

	log.Printf("seeded tenant %s (%s)", t.Name, t.ID)
}

func featureFlag(name string) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New(name + " must be a boolean")
	}
	return enabled, nil
}
