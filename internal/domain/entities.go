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

type TenantSettings struct {
	TenantID           uuid.UUID `json:"tenantId"`
	FeatureDesignAI    bool      `json:"featureDesignAi"`
	DesignAIDailyLimit int       `json:"designAiDailyLimit"`
	DesignAIMaxTotal   int       `json:"designAiMaxTotal"`
}

type Customer struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenantId"`
	ExternalID string     `json:"externalId,omitempty"`
	NationalID *string    `json:"nationalId,omitempty"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	FirstName  string     `json:"firstName"`
	LastName   string     `json:"lastName"`
	Phone      string     `json:"phone"`
	Role       string     `json:"role"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
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

type CatalogService struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenantId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DurationMin int       `json:"durationMin"`
	PriceCents  int       `json:"priceCents"`
	Currency    string    `json:"currency"`
	Active      bool      `json:"active"`
}

type BookingStatus string

const (
	BookingPending    BookingStatus = "PENDING"
	BookingConfirmed  BookingStatus = "CONFIRMED"
	BookingInProgress BookingStatus = "IN_PROGRESS"
	BookingCompleted  BookingStatus = "COMPLETED"
	BookingCancelled  BookingStatus = "CANCELLED"
	BookingNoShow     BookingStatus = "NO_SHOW"
)

type OperativeStage string

const (
	StageBooked     OperativeStage = "BOOKEADA"
	StageInProgress OperativeStage = "EN_PROGRESO"
	StagePaid       OperativeStage = "PAGADA"
	StageFinished   OperativeStage = "FINALIZADA"
	StageCancelled  OperativeStage = "CANCELADA"
)

type Booking struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenantId"`
	CustomerID     uuid.UUID      `json:"customerId"`
	ServiceID      uuid.UUID      `json:"serviceId"`
	DesignID       *uuid.UUID     `json:"designId,omitempty"`
	StartsAt       time.Time      `json:"startsAt"`
	Status         BookingStatus  `json:"status"`
	OperativeStage OperativeStage `json:"operativeStage"`
	Notes          string         `json:"notes"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type PaymentMethod string

const (
	PaymentCash     PaymentMethod = "CASH"
	PaymentTransfer PaymentMethod = "TRANSFER"
	PaymentCard     PaymentMethod = "CARD"
)

type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "PENDING"
	PaymentReported PaymentStatus = "REPORTED"
	PaymentPaid     PaymentStatus = "PAID"
	PaymentVoided   PaymentStatus = "VOIDED"
)

type Payment struct {
	ID            uuid.UUID      `json:"id"`
	TenantID      uuid.UUID      `json:"tenantId"`
	CustomerID    uuid.UUID      `json:"customerId"`
	BookingID     *uuid.UUID     `json:"bookingId,omitempty"`
	BankAccountID *uuid.UUID     `json:"bankAccountId,omitempty"`
	BankCode      string         `json:"bankCode"`
	Method        PaymentMethod  `json:"method"`
	Status        PaymentStatus  `json:"status"`
	AmountCents   int            `json:"amountCents"`
	Currency      string         `json:"currency"`
	Reference     string         `json:"reference"`
	ReceiptURL    string         `json:"receiptUrl"`
	AdminNotes    string         `json:"adminNotes"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type BankAccount struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenantId"`
	BankCode      string    `json:"bankCode"`
	HolderName    string    `json:"holderName"`
	AccountNumber string    `json:"accountNumber"`
	AccountType   string    `json:"accountType"`
	TaxID         string    `json:"taxId"`
	NotifyEmail   string    `json:"notifyEmail"`
	Active        bool      `json:"active"`
	SortOrder     int       `json:"sortOrder"`
}

type DesignStatus string

const (
	DesignPending    DesignStatus = "PENDIENTE"
	DesignProcessing DesignStatus = "PROCESANDO"
	DesignReady      DesignStatus = "LISTO"
	DesignFailed     DesignStatus = "FALLIDO"
	DesignUsed       DesignStatus = "USADO_EN_CITA"
)

type Design struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenantId"`
	CustomerID   uuid.UUID    `json:"customerId"`
	Prompt       string       `json:"prompt"`
	PhotoURL     string       `json:"photoUrl"`
	ResultURL    string       `json:"resultUrl"`
	ModelUsed    string       `json:"modelUsed"`
	Status       DesignStatus `json:"status"`
	ErrorMessage string       `json:"errorMessage"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	ProcessedAt  *time.Time   `json:"processedAt,omitempty"`
}
