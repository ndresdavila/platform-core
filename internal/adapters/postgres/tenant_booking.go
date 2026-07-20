package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndresdavila/platform-core/internal/domain"
)

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

type SettingsRepo struct{ db *pgxpool.Pool }

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo { return &SettingsRepo{db: db} }

func (r *SettingsRepo) Get(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error) {
	row := r.db.QueryRow(ctx, `
		SELECT tenant_id, feature_design_ai, design_ai_daily_limit, design_ai_max_total
		FROM tenant_settings WHERE tenant_id = $1`, tenantID)
	var s domain.TenantSettings
	err := row.Scan(&s.TenantID, &s.FeatureDesignAI, &s.DesignAIDailyLimit, &s.DesignAIMaxTotal)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.TenantSettings{
			TenantID: tenantID, FeatureDesignAI: false,
			DesignAIDailyLimit: 3, DesignAIMaxTotal: 5,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingsRepo) Upsert(ctx context.Context, s *domain.TenantSettings) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenant_settings (tenant_id, feature_design_ai, design_ai_daily_limit, design_ai_max_total)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (tenant_id) DO UPDATE SET
		  feature_design_ai = EXCLUDED.feature_design_ai,
		  design_ai_daily_limit = EXCLUDED.design_ai_daily_limit,
		  design_ai_max_total = EXCLUDED.design_ai_max_total`,
		s.TenantID, s.FeatureDesignAI, s.DesignAIDailyLimit, s.DesignAIMaxTotal)
	return err
}

type scannable interface{ Scan(dest ...any) error }

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

type BookingRepo struct{ db *pgxpool.Pool }

func NewBookingRepo(db *pgxpool.Pool) *BookingRepo { return &BookingRepo{db: db} }

func (r *BookingRepo) Create(ctx context.Context, b *domain.Booking) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO bookings (id, tenant_id, customer_id, service_id, design_id, starts_at, status, operative_stage, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		b.ID, b.TenantID, b.CustomerID, b.ServiceID, b.DesignID, b.StartsAt, b.Status, b.OperativeStage, b.Notes, b.CreatedAt, b.UpdatedAt)
	return err
}

func (r *BookingRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Booking, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, service_id, design_id, starts_at, status, operative_stage, notes, created_at, updated_at
		FROM bookings WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return scanBooking(row)
}

func (r *BookingRepo) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Booking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, customer_id, service_id, design_id, starts_at, status, operative_stage, notes, created_at, updated_at
		FROM bookings WHERE tenant_id = $1 AND customer_id = $2 ORDER BY starts_at DESC`, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookings(rows)
}

func (r *BookingRepo) ListByRange(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.Booking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, customer_id, service_id, design_id, starts_at, status, operative_stage, notes, created_at, updated_at
		FROM bookings WHERE tenant_id = $1 AND starts_at >= $2 AND starts_at <= $3 ORDER BY starts_at ASC`,
		tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookings(rows)
}

func (r *BookingRepo) Update(ctx context.Context, b *domain.Booking) error {
	_, err := r.db.Exec(ctx, `
		UPDATE bookings SET status=$3, operative_stage=$4, notes=$5, design_id=$6, updated_at=$7
		WHERE tenant_id=$1 AND id=$2`,
		b.TenantID, b.ID, b.Status, b.OperativeStage, b.Notes, b.DesignID, b.UpdatedAt)
	return err
}

type rowsq interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func collectBookings(rows rowsq) ([]domain.Booking, error) {
	var out []domain.Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func scanBooking(row scannable) (*domain.Booking, error) {
	var b domain.Booking
	err := row.Scan(&b.ID, &b.TenantID, &b.CustomerID, &b.ServiceID, &b.DesignID, &b.StartsAt, &b.Status, &b.OperativeStage, &b.Notes, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}
