package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}

type Employee struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenantId"`
	ExternalID string    `json:"externalId"`
	Email      string    `json:"email"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Role       string    `json:"role"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// --- Facturación electrónica (Masterview MVP) ---

type Person struct {
	ID                 uuid.UUID `json:"id"`
	TenantID           uuid.UUID `json:"tenantId"`
	Kind               string    `json:"kind"`
	IdentificationType string    `json:"identificationType"`
	Identification     string    `json:"identification"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Phone              string    `json:"phone"`
	Address            string    `json:"address"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Product struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenantId"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Unit       string    `json:"unit"`
	PriceCents int       `json:"priceCents"`
	IVARate    float64   `json:"ivaRate"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ElectronicDocumentStatus string

const (
	DocSaved        ElectronicDocumentStatus = "SAVED"
	DocAuthorized   ElectronicDocumentStatus = "AUTHORIZED"
)

type ElectronicDocumentLine struct {
	ID              uuid.UUID  `json:"id"`
	DocumentID      uuid.UUID  `json:"documentId"`
	TenantID        uuid.UUID  `json:"tenantId"`
	LineNo          int        `json:"lineNo"`
	ProductID       *uuid.UUID `json:"productId,omitempty"`
	ProductName     string     `json:"productName"`
	Unit            string     `json:"unit"`
	Quantity        float64    `json:"quantity"`
	UnitPriceCents  int        `json:"unitPriceCents"`
	IVARate         float64    `json:"ivaRate"`
	DiscountPercent float64    `json:"discountPercent"`
	DiscountCents   int        `json:"discountCents"`
	SubtotalCents   int        `json:"subtotalCents"`
}

type ElectronicDocument struct {
	ID                   uuid.UUID                `json:"id"`
	TenantID             uuid.UUID                `json:"tenantId"`
	DocType              string                   `json:"docType"`
	PartyKind            string                   `json:"partyKind"`
	PersonID             *uuid.UUID               `json:"personId,omitempty"`
	PersonName           string                   `json:"personName"`
	PersonIdentification string                   `json:"personIdentification"`
	Establishment        string                   `json:"establishment"`
	EmissionPoint        string                   `json:"emissionPoint"`
	DocumentNumber       string                   `json:"documentNumber"`
	AccessKey            string                   `json:"accessKey"`
	IssueDate            time.Time                `json:"issueDate"`
	DueDays              int                      `json:"dueDays"`
	Reference            string                   `json:"reference"`
	Seller               string                   `json:"seller"`
	Description          string                   `json:"description"`
	IsExport             bool                     `json:"isExport"`
	Status               ElectronicDocumentStatus `json:"status"`
	SRIMessage           string                   `json:"sriMessage"`
	Subtotal15Cents      int                      `json:"subtotal15Cents"`
	Subtotal5Cents       int                      `json:"subtotal5Cents"`
	Subtotal0Cents       int                      `json:"subtotal0Cents"`
	DiscountCents        int                      `json:"discountCents"`
	IVA15Cents           int                      `json:"iva15Cents"`
	IVA5Cents            int                      `json:"iva5Cents"`
	ICECents             int                      `json:"iceCents"`
	TotalCents           int                      `json:"totalCents"`
	CreatedByExternalID  string                   `json:"createdByExternalId"`
	CreatedAt            time.Time                `json:"createdAt"`
	UpdatedAt            time.Time                `json:"updatedAt"`
	Lines                []ElectronicDocumentLine `json:"lines,omitempty"`
}
