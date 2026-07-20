package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type CustomerRepo struct{ db *pgxpool.Pool }

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo { return &CustomerRepo{db: db} }

func scanCustomer(row scannable) (*domain.Customer, error) {
	var c domain.Customer
	var national *string
	err := row.Scan(&c.ID, &c.TenantID, &c.ExternalID, &national, &c.Email, &c.Name, &c.FirstName, &c.LastName, &c.Phone, &c.Role, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.NationalID = national
	return &c, nil
}

const customerCols = `id, tenant_id, COALESCE(external_id,''), national_id, email, name, COALESCE(first_name,''), COALESCE(last_name,''), phone, COALESCE(role,'CLIENTE'), COALESCE(active,true), created_at, COALESCE(updated_at, created_at)`

func (r *CustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	row := r.db.QueryRow(ctx, `SELECT `+customerCols+` FROM customers WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return scanCustomer(row)
}

func (r *CustomerRepo) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Customer, error) {
	row := r.db.QueryRow(ctx, `SELECT `+customerCols+` FROM customers WHERE tenant_id=$1 AND external_id=$2`, tenantID, externalID)
	return scanCustomer(row)
}

func (r *CustomerRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Customer, error) {
	row := r.db.QueryRow(ctx, `SELECT `+customerCols+` FROM customers WHERE tenant_id=$1 AND lower(email)=lower($2)`, tenantID, email)
	return scanCustomer(row)
}

func (r *CustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customers (id, tenant_id, external_id, national_id, email, name, first_name, last_name, phone, role, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		c.ID, c.TenantID, c.ExternalID, c.NationalID, c.Email, c.Name, c.FirstName, c.LastName, c.Phone, c.Role, c.Active, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CustomerRepo) Update(ctx context.Context, c *domain.Customer) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers SET external_id=$3, national_id=$4, email=$5, name=$6, first_name=$7, last_name=$8,
		phone=$9, role=$10, active=$11, updated_at=$12 WHERE tenant_id=$1 AND id=$2`,
		c.TenantID, c.ID, c.ExternalID, c.NationalID, c.Email, c.Name, c.FirstName, c.LastName, c.Phone, c.Role, c.Active, c.UpdatedAt)
	return err
}

func (r *CustomerRepo) UpsertByExternalID(ctx context.Context, c *domain.Customer) error {
	mergeExisting := func(existing *domain.Customer) error {
		c.ID = existing.ID
		c.CreatedAt = existing.CreatedAt
		// Never wipe profile fields when the caller omits them (auth sync).
		if c.NationalID == nil || (c.NationalID != nil && *c.NationalID == "") {
			c.NationalID = existing.NationalID
		}
		if c.Phone == "" {
			c.Phone = existing.Phone
		}
		if c.FirstName == "" {
			c.FirstName = existing.FirstName
		}
		if c.LastName == "" {
			c.LastName = existing.LastName
		}
		if c.Name == "" || strings.TrimSpace(c.Name) == "" {
			c.Name = existing.Name
		}
		if c.Role == "" {
			c.Role = existing.Role
		}
		return r.Update(ctx, c)
	}

	if c.ExternalID != "" {
		if existing, err := r.GetByExternalID(ctx, c.TenantID, c.ExternalID); err == nil {
			return mergeExisting(existing)
		}
	}
	if existing, err := r.GetByEmail(ctx, c.TenantID, c.Email); err == nil {
		if c.ExternalID != "" && existing.ExternalID == "" {
			existing.ExternalID = c.ExternalID
		}
		return mergeExisting(existing)
	}
	return r.Create(ctx, c)
}

type EmployeeRepo struct{ db *pgxpool.Pool }

func NewEmployeeRepo(db *pgxpool.Pool) *EmployeeRepo { return &EmployeeRepo{db: db} }

func (r *EmployeeRepo) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Employee, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, external_id, email, first_name, last_name, role, active, created_at, updated_at
		FROM employees WHERE tenant_id=$1 AND external_id=$2`, tenantID, externalID)
	return scanEmployee(row)
}

func (r *EmployeeRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Employee, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, external_id, email, first_name, last_name, role, active, created_at, updated_at
		FROM employees WHERE tenant_id=$1 AND lower(email)=lower($2)`, tenantID, email)
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

type CatalogRepo struct{ db *pgxpool.Pool }

func NewCatalogRepo(db *pgxpool.Pool) *CatalogRepo { return &CatalogRepo{db: db} }

func (r *CatalogRepo) ListServices(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]domain.CatalogService, error) {
	q := `SELECT id, tenant_id, name, description, duration_min, price_cents, currency, active FROM catalog_services WHERE tenant_id=$1`
	if activeOnly {
		q += ` AND active=true`
	}
	q += ` ORDER BY name ASC`
	rows, err := r.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogService
	for rows.Next() {
		var s domain.CatalogService
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.DurationMin, &s.PriceCents, &s.Currency, &s.Active); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *CatalogRepo) GetService(ctx context.Context, tenantID, id uuid.UUID) (*domain.CatalogService, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, duration_min, price_cents, currency, active
		FROM catalog_services WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	var s domain.CatalogService
	err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.DurationMin, &s.PriceCents, &s.Currency, &s.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &s, err
}

func (r *CatalogRepo) CreateService(ctx context.Context, s *domain.CatalogService) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO catalog_services (id, tenant_id, name, description, duration_min, price_cents, currency, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		s.ID, s.TenantID, s.Name, s.Description, s.DurationMin, s.PriceCents, s.Currency, s.Active)
	return err
}

type PaymentRepo struct{ db *pgxpool.Pool }

func NewPaymentRepo(db *pgxpool.Pool) *PaymentRepo { return &PaymentRepo{db: db} }

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO payments (id, tenant_id, customer_id, booking_id, bank_account_id, bank_code, method, status, amount_cents, currency, reference, receipt_url, admin_notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		p.ID, p.TenantID, p.CustomerID, p.BookingID, p.BankAccountID, p.BankCode, p.Method, p.Status, p.AmountCents, p.Currency, p.Reference, p.ReceiptURL, p.AdminNotes, p.CreatedAt, p.UpdatedAt)
	return err
}

func paymentSelect() string {
	return `id, tenant_id, customer_id, booking_id, bank_account_id, COALESCE(bank_code,''), method, status, amount_cents, currency, reference, receipt_url, COALESCE(admin_notes,''), created_at, updated_at`
}

func (r *PaymentRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	row := r.db.QueryRow(ctx, `SELECT `+paymentSelect()+` FROM payments WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return scanPayment(row)
}

func (r *PaymentRepo) GetLatestByBooking(ctx context.Context, tenantID, bookingID uuid.UUID) (*domain.Payment, error) {
	row := r.db.QueryRow(ctx, `SELECT `+paymentSelect()+` FROM payments WHERE tenant_id=$1 AND booking_id=$2 ORDER BY created_at DESC LIMIT 1`, tenantID, bookingID)
	return scanPayment(row)
}

func (r *PaymentRepo) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Payment, error) {
	rows, err := r.db.Query(ctx, `SELECT `+paymentSelect()+` FROM payments WHERE tenant_id=$1 AND customer_id=$2 ORDER BY created_at DESC`, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.Exec(ctx, `
		UPDATE payments SET status=$3, reference=$4, receipt_url=$5, admin_notes=$6, bank_code=$7, bank_account_id=$8, updated_at=$9
		WHERE tenant_id=$1 AND id=$2`,
		p.TenantID, p.ID, p.Status, p.Reference, p.ReceiptURL, p.AdminNotes, p.BankCode, p.BankAccountID, p.UpdatedAt)
	return err
}

func scanPayment(row scannable) (*domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(&p.ID, &p.TenantID, &p.CustomerID, &p.BookingID, &p.BankAccountID, &p.BankCode, &p.Method, &p.Status, &p.AmountCents, &p.Currency, &p.Reference, &p.ReceiptURL, &p.AdminNotes, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type BankAccountRepo struct{ db *pgxpool.Pool }

func NewBankAccountRepo(db *pgxpool.Pool) *BankAccountRepo { return &BankAccountRepo{db: db} }

func (r *BankAccountRepo) ListActive(ctx context.Context, tenantID uuid.UUID) ([]domain.BankAccount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, bank_code, holder_name, account_number, account_type, tax_id, notify_email, active, sort_order
		FROM bank_accounts WHERE tenant_id=$1 AND active=true ORDER BY sort_order ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BankAccount
	for rows.Next() {
		var a domain.BankAccount
		if err := rows.Scan(&a.ID, &a.TenantID, &a.BankCode, &a.HolderName, &a.AccountNumber, &a.AccountType, &a.TaxID, &a.NotifyEmail, &a.Active, &a.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *BankAccountRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BankAccount, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, bank_code, holder_name, account_number, account_type, tax_id, notify_email, active, sort_order
		FROM bank_accounts WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	var a domain.BankAccount
	err := row.Scan(&a.ID, &a.TenantID, &a.BankCode, &a.HolderName, &a.AccountNumber, &a.AccountType, &a.TaxID, &a.NotifyEmail, &a.Active, &a.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &a, err
}

func (r *BankAccountRepo) Create(ctx context.Context, a *domain.BankAccount) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO bank_accounts (id, tenant_id, bank_code, holder_name, account_number, account_type, tax_id, notify_email, active, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		a.ID, a.TenantID, a.BankCode, a.HolderName, a.AccountNumber, a.AccountType, a.TaxID, a.NotifyEmail, a.Active, a.SortOrder)
	return err
}

type DesignRepo struct{ db *pgxpool.Pool }

func NewDesignRepo(db *pgxpool.Pool) *DesignRepo { return &DesignRepo{db: db} }

func (r *DesignRepo) Create(ctx context.Context, d *domain.Design) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO designs (id, tenant_id, customer_id, prompt, photo_url, result_url, model_used, status, error_message, created_at, updated_at, processed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		d.ID, d.TenantID, d.CustomerID, d.Prompt, d.PhotoURL, d.ResultURL, d.ModelUsed, d.Status, d.ErrorMessage, d.CreatedAt, d.UpdatedAt, d.ProcessedAt)
	return err
}

func (r *DesignRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Design, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, customer_id, prompt, photo_url, result_url, model_used, status, error_message, created_at, updated_at, processed_at
		FROM designs WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return scanDesign(row)
}

func (r *DesignRepo) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Design, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, customer_id, prompt, photo_url, result_url, model_used, status, error_message, created_at, updated_at, processed_at
		FROM designs WHERE tenant_id=$1 AND customer_id=$2 ORDER BY created_at DESC`, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Design
	for rows.Next() {
		d, err := scanDesign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DesignRepo) Update(ctx context.Context, d *domain.Design) error {
	_, err := r.db.Exec(ctx, `
		UPDATE designs SET prompt=$3, photo_url=$4, result_url=$5, model_used=$6, status=$7, error_message=$8, updated_at=$9, processed_at=$10
		WHERE tenant_id=$1 AND id=$2`,
		d.TenantID, d.ID, d.Prompt, d.PhotoURL, d.ResultURL, d.ModelUsed, d.Status, d.ErrorMessage, d.UpdatedAt, d.ProcessedAt)
	return err
}

func (r *DesignRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM designs WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	return err
}

func (r *DesignRepo) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM designs WHERE tenant_id=$1 AND customer_id=$2`, tenantID, customerID).Scan(&n)
	return n, err
}

func (r *DesignRepo) CountByCustomerSince(ctx context.Context, tenantID, customerID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM designs WHERE tenant_id=$1 AND customer_id=$2 AND created_at >= $3`, tenantID, customerID, since).Scan(&n)
	return n, err
}

func scanDesign(row scannable) (*domain.Design, error) {
	var d domain.Design
	err := row.Scan(&d.ID, &d.TenantID, &d.CustomerID, &d.Prompt, &d.PhotoURL, &d.ResultURL, &d.ModelUsed, &d.Status, &d.ErrorMessage, &d.CreatedAt, &d.UpdatedAt, &d.ProcessedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}
