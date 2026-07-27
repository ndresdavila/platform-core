package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type PersonRepo struct{ db *pgxpool.Pool }

func NewPersonRepo(db *pgxpool.Pool) *PersonRepo { return &PersonRepo{db: db} }

func (r *PersonRepo) Create(ctx context.Context, p *domain.Person) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO persons (id, tenant_id, kind, identification_type, identification, name, email, phone, address, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		p.ID, p.TenantID, p.Kind, p.IdentificationType, p.Identification, p.Name, p.Email, p.Phone, p.Address, p.Active, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PersonRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Person, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, kind, identification_type, identification, name, email, phone, address, active, created_at, updated_at
		FROM persons WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	var p domain.Person
	err := row.Scan(&p.ID, &p.TenantID, &p.Kind, &p.IdentificationType, &p.Identification, &p.Name, &p.Email, &p.Phone, &p.Address, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *PersonRepo) List(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Person, error) {
	q = strings.TrimSpace(q)
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, kind, identification_type, identification, name, email, phone, address, active, created_at, updated_at
		FROM persons
		WHERE tenant_id=$1 AND active=true
		  AND ($2='' OR name ILIKE '%'||$2||'%' OR identification ILIKE '%'||$2||'%')
		ORDER BY name ASC
		LIMIT 100`, tenantID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Person
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Kind, &p.IdentificationType, &p.Identification, &p.Name, &p.Email, &p.Phone, &p.Address, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type ProductRepo struct{ db *pgxpool.Pool }

func NewProductRepo(db *pgxpool.Pool) *ProductRepo { return &ProductRepo{db: db} }

func (r *ProductRepo) Create(ctx context.Context, p *domain.Product) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO products (id, tenant_id, code, name, unit, price_cents, iva_rate, active, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.TenantID, p.Code, p.Name, p.Unit, p.PriceCents, p.IVARate, p.Active, p.CreatedAt)
	return err
}

func (r *ProductRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, code, name, unit, price_cents, iva_rate, active, created_at
		FROM products WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	var p domain.Product
	err := row.Scan(&p.ID, &p.TenantID, &p.Code, &p.Name, &p.Unit, &p.PriceCents, &p.IVARate, &p.Active, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *ProductRepo) List(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Product, error) {
	q = strings.TrimSpace(q)
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, code, name, unit, price_cents, iva_rate, active, created_at
		FROM products
		WHERE tenant_id=$1 AND active=true
		  AND ($2='' OR name ILIKE '%'||$2||'%' OR code ILIKE '%'||$2||'%')
		ORDER BY name ASC
		LIMIT 100`, tenantID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Code, &p.Name, &p.Unit, &p.PriceCents, &p.IVARate, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type InvoiceRepo struct{ db *pgxpool.Pool }

func NewInvoiceRepo(db *pgxpool.Pool) *InvoiceRepo { return &InvoiceRepo{db: db} }

func (r *InvoiceRepo) NextDocumentNumber(ctx context.Context, tenantID uuid.UUID, establishment, emissionPoint, docType string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var next int64
	err = tx.QueryRow(ctx, `
		INSERT INTO document_sequences (tenant_id, establishment, emission_point, doc_type, next_number)
		VALUES ($1,$2,$3,$4,2)
		ON CONFLICT (tenant_id, establishment, emission_point, doc_type)
		DO UPDATE SET next_number = document_sequences.next_number + 1
		RETURNING next_number - 1`, tenantID, establishment, emissionPoint, docType).Scan(&next)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%09d", establishment, emissionPoint, next), nil
}

func (r *InvoiceRepo) Create(ctx context.Context, doc *domain.ElectronicDocument) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO electronic_documents (
			id, tenant_id, doc_type, party_kind, person_id, person_name, person_identification,
			establishment, emission_point, document_number, access_key, issue_date, due_days,
			reference, seller, description, is_export, status, sri_message,
			subtotal_15_cents, subtotal_5_cents, subtotal_0_cents, discount_cents,
			iva_15_cents, iva_5_cents, ice_cents, total_cents, created_by_external_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,
			$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30
		)`,
		doc.ID, doc.TenantID, doc.DocType, doc.PartyKind, doc.PersonID, doc.PersonName, doc.PersonIdentification,
		doc.Establishment, doc.EmissionPoint, doc.DocumentNumber, doc.AccessKey, doc.IssueDate, doc.DueDays,
		doc.Reference, doc.Seller, doc.Description, doc.IsExport, doc.Status, doc.SRIMessage,
		doc.Subtotal15Cents, doc.Subtotal5Cents, doc.Subtotal0Cents, doc.DiscountCents,
		doc.IVA15Cents, doc.IVA5Cents, doc.ICECents, doc.TotalCents, doc.CreatedByExternalID, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return err
	}
	for _, line := range doc.Lines {
		_, err = tx.Exec(ctx, `
			INSERT INTO electronic_document_lines (
				id, document_id, tenant_id, line_no, product_id, product_name, unit, quantity,
				unit_price_cents, iva_rate, discount_percent, discount_cents, subtotal_cents
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			line.ID, doc.ID, doc.TenantID, line.LineNo, line.ProductID, line.ProductName, line.Unit, line.Quantity,
			line.UnitPriceCents, line.IVARate, line.DiscountPercent, line.DiscountCents, line.SubtotalCents,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *InvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ElectronicDocument, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, doc_type, party_kind, person_id, person_name, person_identification,
			establishment, emission_point, document_number, access_key, issue_date, due_days,
			reference, seller, description, is_export, status, sri_message,
			subtotal_15_cents, subtotal_5_cents, subtotal_0_cents, discount_cents,
			iva_15_cents, iva_5_cents, ice_cents, total_cents, created_by_external_id, created_at, updated_at
		FROM electronic_documents WHERE tenant_id=$1 AND id=$2`, tenantID, id)
	doc, err := scanDocument(row)
	if err != nil {
		return nil, err
	}
	lines, err := r.listLines(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	doc.Lines = lines
	return doc, nil
}

func (r *InvoiceRepo) List(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ElectronicDocument, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, doc_type, party_kind, person_id, person_name, person_identification,
			establishment, emission_point, document_number, access_key, issue_date, due_days,
			reference, seller, description, is_export, status, sri_message,
			subtotal_15_cents, subtotal_5_cents, subtotal_0_cents, discount_cents,
			iva_15_cents, iva_5_cents, ice_cents, total_cents, created_by_external_id, created_at, updated_at
		FROM electronic_documents WHERE tenant_id=$1
		ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ElectronicDocument
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *doc)
	}
	return out, rows.Err()
}

func (r *InvoiceRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ElectronicDocumentStatus, sriMessage, accessKey string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE electronic_documents
		SET status=$3, sri_message=$4, access_key=CASE WHEN $5='' THEN access_key ELSE $5 END, updated_at=$6
		WHERE tenant_id=$1 AND id=$2`, tenantID, id, status, sriMessage, accessKey, time.Now().UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *InvoiceRepo) listLines(ctx context.Context, tenantID, documentID uuid.UUID) ([]domain.ElectronicDocumentLine, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, document_id, tenant_id, line_no, product_id, product_name, unit, quantity,
			unit_price_cents, iva_rate, discount_percent, discount_cents, subtotal_cents
		FROM electronic_document_lines WHERE tenant_id=$1 AND document_id=$2
		ORDER BY line_no ASC`, tenantID, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ElectronicDocumentLine
	for rows.Next() {
		var l domain.ElectronicDocumentLine
		if err := rows.Scan(&l.ID, &l.DocumentID, &l.TenantID, &l.LineNo, &l.ProductID, &l.ProductName, &l.Unit, &l.Quantity,
			&l.UnitPriceCents, &l.IVARate, &l.DiscountPercent, &l.DiscountCents, &l.SubtotalCents); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanDocument(row scannable) (*domain.ElectronicDocument, error) {
	var d domain.ElectronicDocument
	var status string
	err := row.Scan(
		&d.ID, &d.TenantID, &d.DocType, &d.PartyKind, &d.PersonID, &d.PersonName, &d.PersonIdentification,
		&d.Establishment, &d.EmissionPoint, &d.DocumentNumber, &d.AccessKey, &d.IssueDate, &d.DueDays,
		&d.Reference, &d.Seller, &d.Description, &d.IsExport, &status, &d.SRIMessage,
		&d.Subtotal15Cents, &d.Subtotal5Cents, &d.Subtotal0Cents, &d.DiscountCents,
		&d.IVA15Cents, &d.IVA5Cents, &d.ICECents, &d.TotalCents, &d.CreatedByExternalID, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.Status = domain.ElectronicDocumentStatus(status)
	return &d, nil
}
