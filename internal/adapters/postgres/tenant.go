package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type scannable interface{ Scan(dest ...any) error }

type TenantRepo struct{ db *pgxpool.Pool }

func NewTenantRepo(db *pgxpool.Pool) *TenantRepo { return &TenantRepo{db: db} }

func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	row := r.db.QueryRow(ctx, `SELECT id, slug, name, active, created_at FROM tenants WHERE id = $1`, id)
	return scanTenant(row)
}

func (r *TenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	row := r.db.QueryRow(ctx, `SELECT id, slug, name, active, created_at FROM tenants WHERE slug = $1`, slug)
	return scanTenant(row)
}

func (r *TenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenants (id, slug, name, active, created_at) VALUES ($1,$2,$3,$4,$5)`,
		t.ID, t.Slug, t.Name, t.Active, t.CreatedAt)
	return err
}

func scanTenant(row scannable) (*domain.Tenant, error) {
	var t domain.Tenant
	err := row.Scan(&t.ID, &t.Slug, &t.Name, &t.Active, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type EmployeeRepo struct{ db *pgxpool.Pool }

func NewEmployeeRepo(db *pgxpool.Pool) *EmployeeRepo { return &EmployeeRepo{db: db} }

func (r *EmployeeRepo) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Employee, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, external_id, email, first_name, last_name, role, active, created_at, updated_at
		FROM employees WHERE tenant_id=$1 AND external_id=$2`, tenantID, externalID)
	return scanEmployee(row)
}

func (r *EmployeeRepo) UpsertByExternalID(ctx context.Context, e *domain.Employee) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employees (id, tenant_id, external_id, email, first_name, last_name, role, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (tenant_id, external_id) DO UPDATE SET
		  email=EXCLUDED.email, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
		  role=EXCLUDED.role, updated_at=EXCLUDED.updated_at`,
		e.ID, e.TenantID, e.ExternalID, e.Email, e.FirstName, e.LastName, e.Role, e.Active, e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return err
	}
	got, err := r.GetByExternalID(ctx, e.TenantID, e.ExternalID)
	if err == nil {
		e.ID = got.ID
	}
	return err
}

func scanEmployee(row scannable) (*domain.Employee, error) {
	var e domain.Employee
	err := row.Scan(&e.ID, &e.TenantID, &e.ExternalID, &e.Email, &e.FirstName, &e.LastName, &e.Role, &e.Active, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
