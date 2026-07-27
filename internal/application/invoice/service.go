package invoice

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants  ports.TenantRepository
	persons  ports.PersonRepository
	products ports.ProductRepository
	invoices ports.InvoiceRepository
	clock    ports.Clock
	ids      ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	persons ports.PersonRepository,
	products ports.ProductRepository,
	invoices ports.InvoiceRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{tenants: tenants, persons: persons, products: products, invoices: invoices, clock: clock, ids: ids}
}

type CreatePersonInput struct {
	TenantID           uuid.UUID `json:"-"`
	Kind               string    `json:"kind"`
	IdentificationType string    `json:"identificationType"`
	Identification     string    `json:"identification"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Phone              string    `json:"phone"`
	Address            string    `json:"address"`
}

func (s *Service) CreatePerson(ctx context.Context, in CreatePersonInput) (*domain.Person, error) {
	if err := s.requireTenant(ctx, in.TenantID); err != nil {
		return nil, err
	}
	in.Identification = strings.TrimSpace(in.Identification)
	in.Name = strings.TrimSpace(in.Name)
	if in.Identification == "" || in.Name == "" {
		return nil, domain.ErrValidation
	}
	if in.Kind == "" {
		in.Kind = "CLIENTE"
	}
	if in.IdentificationType == "" {
		in.IdentificationType = "04"
	}
	now := s.clock.Now()
	p := &domain.Person{
		ID: s.ids.New(), TenantID: in.TenantID, Kind: in.Kind,
		IdentificationType: in.IdentificationType, Identification: in.Identification,
		Name: in.Name, Email: in.Email, Phone: in.Phone, Address: in.Address,
		Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.persons.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) ListPersons(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Person, error) {
	if err := s.requireTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.persons.List(ctx, tenantID, q)
}

type CreateProductInput struct {
	TenantID   uuid.UUID `json:"-"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Unit       string    `json:"unit"`
	PriceCents int       `json:"priceCents"`
	IVARate    float64   `json:"ivaRate"`
}

func (s *Service) CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	if err := s.requireTenant(ctx, in.TenantID); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, domain.ErrValidation
	}
	if in.Unit == "" {
		in.Unit = "UND"
	}
	if in.Code == "" {
		in.Code = strings.ToUpper(strings.ReplaceAll(in.Name, " ", "-"))
	}
	if in.IVARate == 0 {
		in.IVARate = 15
	}
	p := &domain.Product{
		ID: s.ids.New(), TenantID: in.TenantID, Code: in.Code, Name: in.Name,
		Unit: in.Unit, PriceCents: in.PriceCents, IVARate: in.IVARate, Active: true, CreatedAt: s.clock.Now(),
	}
	if err := s.products.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) ListProducts(ctx context.Context, tenantID uuid.UUID, q string) ([]domain.Product, error) {
	if err := s.requireTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.products.List(ctx, tenantID, q)
}

type LineInput struct {
	ProductID       *uuid.UUID `json:"productId"`
	ProductName     string     `json:"productName"`
	Unit            string     `json:"unit"`
	Quantity        float64    `json:"quantity"`
	UnitPriceCents  int        `json:"unitPriceCents"`
	IVARate         float64    `json:"ivaRate"`
	DiscountPercent float64    `json:"discountPercent"`
}

type CreateDocumentInput struct {
	TenantID            uuid.UUID
	DocType             string
	PartyKind           string
	PersonID            *uuid.UUID
	PersonName          string
	PersonIdentification string
	Establishment       string
	EmissionPoint       string
	IssueDate           time.Time
	DueDays             int
	Reference           string
	Seller              string
	Description         string
	IsExport            bool
	SendToSRI           bool
	CreatedByExternalID string
	Lines               []LineInput
}

func (s *Service) CreateDocument(ctx context.Context, in CreateDocumentInput) (*domain.ElectronicDocument, error) {
	if err := s.requireTenant(ctx, in.TenantID); err != nil {
		return nil, err
	}
	if len(in.Lines) == 0 {
		return nil, domain.ErrValidation
	}
	if strings.TrimSpace(in.Description) == "" {
		return nil, domain.ErrValidation
	}
	if in.Establishment == "" {
		in.Establishment = "001"
	}
	if in.EmissionPoint == "" {
		in.EmissionPoint = "001"
	}
	if in.DocType == "" {
		in.DocType = "FACTURA"
	}
	if in.PartyKind == "" {
		in.PartyKind = "CLIENTE"
	}
	if in.IssueDate.IsZero() {
		in.IssueDate = s.clock.Now()
	}

	if in.PersonID != nil {
		p, err := s.persons.GetByID(ctx, in.TenantID, *in.PersonID)
		if err != nil {
			return nil, err
		}
		in.PersonName = p.Name
		in.PersonIdentification = p.Identification
	}
	if strings.TrimSpace(in.PersonName) == "" {
		return nil, domain.ErrValidation
	}

	number, err := s.invoices.NextDocumentNumber(ctx, in.TenantID, in.Establishment, in.EmissionPoint, in.DocType)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	doc := &domain.ElectronicDocument{
		ID: s.ids.New(), TenantID: in.TenantID, DocType: in.DocType, PartyKind: in.PartyKind,
		PersonID: in.PersonID, PersonName: in.PersonName, PersonIdentification: in.PersonIdentification,
		Establishment: in.Establishment, EmissionPoint: in.EmissionPoint, DocumentNumber: number,
		IssueDate: in.IssueDate, DueDays: in.DueDays, Reference: in.Reference, Seller: in.Seller,
		Description: in.Description, IsExport: in.IsExport, Status: domain.DocSaved,
		CreatedByExternalID: in.CreatedByExternalID, CreatedAt: now, UpdatedAt: now,
	}

	for i, raw := range in.Lines {
		line, totals := buildLine(s.ids.New(), doc.ID, in.TenantID, i+1, raw)
		doc.Lines = append(doc.Lines, line)
		doc.DiscountCents += totals.discount
		switch {
		case almostEq(line.IVARate, 15):
			doc.Subtotal15Cents += totals.net
			doc.IVA15Cents += totals.iva
		case almostEq(line.IVARate, 5):
			doc.Subtotal5Cents += totals.net
			doc.IVA5Cents += totals.iva
		default:
			doc.Subtotal0Cents += totals.net
		}
	}
	doc.TotalCents = doc.Subtotal15Cents + doc.Subtotal5Cents + doc.Subtotal0Cents - doc.DiscountCents + doc.IVA15Cents + doc.IVA5Cents + doc.ICECents

	if in.SendToSRI {
		// MVP: simula envío/autorización local (sin web service SRI real todavía).
		doc.Status = domain.DocAuthorized
		doc.AccessKey = fakeAccessKey(number, now)
		doc.SRIMessage = "AUTORIZADO (simulado — ambiente local MVP)"
	}

	if err := s.invoices.Create(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) GetDocument(ctx context.Context, tenantID, id uuid.UUID) (*domain.ElectronicDocument, error) {
	if err := s.requireTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.invoices.GetByID(ctx, tenantID, id)
}

func (s *Service) ListDocuments(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ElectronicDocument, error) {
	if err := s.requireTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.invoices.List(ctx, tenantID, limit)
}

func (s *Service) SendToSRI(ctx context.Context, tenantID, id uuid.UUID) (*domain.ElectronicDocument, error) {
	if err := s.requireTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	doc, err := s.invoices.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	key := fakeAccessKey(doc.DocumentNumber, s.clock.Now())
	msg := "AUTORIZADO (simulado — ambiente local MVP)"
	if err := s.invoices.UpdateStatus(ctx, tenantID, id, domain.DocAuthorized, msg, key); err != nil {
		return nil, err
	}
	return s.invoices.GetByID(ctx, tenantID, id)
}

func (s *Service) requireTenant(ctx context.Context, tenantID uuid.UUID) error {
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if !t.Active {
		return domain.ErrTenantInactive
	}
	return nil
}

type lineTotals struct{ net, discount, iva int }

func buildLine(id, docID, tenantID uuid.UUID, lineNo int, in LineInput) (domain.ElectronicDocumentLine, lineTotals) {
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	if in.Unit == "" {
		in.Unit = "UND"
	}
	if in.ProductName == "" {
		in.ProductName = "Servicio"
	}
	gross := int(math.Round(in.Quantity * float64(in.UnitPriceCents)))
	discount := int(math.Round(float64(gross) * in.DiscountPercent / 100.0))
	if discount < 0 {
		discount = 0
	}
	net := gross - discount
	iva := int(math.Round(float64(net) * in.IVARate / 100.0))
	line := domain.ElectronicDocumentLine{
		ID: id, DocumentID: docID, TenantID: tenantID, LineNo: lineNo,
		ProductID: in.ProductID, ProductName: in.ProductName, Unit: in.Unit,
		Quantity: in.Quantity, UnitPriceCents: in.UnitPriceCents, IVARate: in.IVARate,
		DiscountPercent: in.DiscountPercent, DiscountCents: discount, SubtotalCents: net,
	}
	return line, lineTotals{net: net, discount: discount, iva: iva}
}

func almostEq(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func fakeAccessKey(documentNumber string, now time.Time) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, documentNumber)
	base := now.Format("02012006") + "01" + "1799999999001" + "1" + digits + "12345678"
	for len(base) < 49 {
		base += "0"
	}
	return base[:49]
}
