package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/ndresdavila/platform-core/internal/adapters/postgres"
	"github.com/ndresdavila/platform-core/internal/application/invoice"
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

	clock := platform.SystemClock{}
	ids := platform.UUIDGen{}
	tenantRepo := postgres.NewTenantRepo(db)
	tenantService := tenant.NewService(tenantRepo, clock, ids)

	t, err := tenantService.GetBySlug(ctx, "masterview")
	if errors.Is(err, domain.ErrNotFound) {
		t, err = tenantService.Create(ctx, tenant.CreateInput{Slug: "masterview", Name: "MASTERVIEW S.A."})
	}
	if err != nil {
		log.Fatalf("ensure tenant: %v", err)
	}

	inv := invoice.NewService(tenantRepo, postgres.NewPersonRepo(db), postgres.NewProductRepo(db), postgres.NewInvoiceRepo(db), clock, ids)

	var personCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM persons WHERE tenant_id=$1`, t.ID).Scan(&personCount); err != nil {
		log.Fatalf("count persons: %v", err)
	}
	if personCount == 0 {
		if _, err := inv.CreatePerson(ctx, invoice.CreatePersonInput{
			TenantID: t.ID, Kind: "CLIENTE", IdentificationType: "04",
			Identification: "1799999999001", Name: "CONSUMIDOR FINAL DEMO",
			Email: "cliente@demo.ec", Phone: "0999999999", Address: "Quito",
		}); err != nil {
			log.Fatalf("seed person: %v", err)
		}
	}

	var productCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE tenant_id=$1`, t.ID).Scan(&productCount); err != nil {
		log.Fatalf("count products: %v", err)
	}
	if productCount == 0 {
		for _, p := range []invoice.CreateProductInput{
			{TenantID: t.ID, Code: "SRV-001", Name: "Servicio profesional", Unit: "UND", PriceCents: 10000, IVARate: 15},
			{TenantID: t.ID, Code: "SRV-002", Name: "Consultoría contable", Unit: "HOR", PriceCents: 25000, IVARate: 15},
			{TenantID: t.ID, Code: "PROD-001", Name: "Producto exento", Unit: "UND", PriceCents: 5000, IVARate: 0},
		} {
			if _, err := inv.CreateProduct(ctx, p); err != nil {
				log.Fatalf("seed product %s: %v", p.Code, err)
			}
		}
	}

	_, _ = db.Exec(ctx, `
		INSERT INTO document_sequences (tenant_id, establishment, emission_point, doc_type, next_number)
		VALUES ($1,'001','001','FACTURA',1)
		ON CONFLICT DO NOTHING`, t.ID)

	log.Printf("seeded tenant %s (%s)", t.Name, t.ID)
}
