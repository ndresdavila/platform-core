package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants  ports.TenantRepository
	payments ports.PaymentRepository
	bookings ports.BookingRepository
	banks    ports.BankAccountRepository
	clock    ports.Clock
	ids      ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	payments ports.PaymentRepository,
	bookings ports.BookingRepository,
	banks ports.BankAccountRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{tenants: tenants, payments: payments, bookings: bookings, banks: banks, clock: clock, ids: ids}
}

type RegisterInput struct {
	TenantID      uuid.UUID
	CustomerID    uuid.UUID
	BookingID     *uuid.UUID
	BankAccountID *uuid.UUID
	BankCode      string
	Method        domain.PaymentMethod
	AmountCents   int
	Currency      string
	Reference     string
	ReceiptURL    string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*domain.Payment, error) {
	t, err := s.tenants.GetByID(ctx, in.TenantID)
	if err != nil {
		return nil, err
	}
	if !t.Active {
		return nil, domain.ErrTenantInactive
	}
	if in.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount_cents must be positive", domain.ErrValidation)
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}
	if in.BookingID != nil {
		if _, err := s.bookings.GetByID(ctx, in.TenantID, *in.BookingID); err != nil {
			return nil, err
		}
	}
	if in.BankAccountID != nil {
		ba, err := s.banks.GetByID(ctx, in.TenantID, *in.BankAccountID)
		if err != nil {
			return nil, err
		}
		in.BankCode = ba.BankCode
	}

	status := domain.PaymentPending
	switch in.Method {
	case domain.PaymentTransfer:
		if in.ReceiptURL == "" {
			return nil, fmt.Errorf("%w: receipt_url required for transfer", domain.ErrValidation)
		}
		status = domain.PaymentReported
	case domain.PaymentCash:
		status = domain.PaymentPending
	case domain.PaymentCard:
		return nil, fmt.Errorf("%w: card payments not enabled", domain.ErrValidation)
	default:
		return nil, fmt.Errorf("%w: unknown payment method", domain.ErrValidation)
	}

	now := s.clock.Now()
	p := &domain.Payment{
		ID:            s.ids.New(),
		TenantID:      in.TenantID,
		CustomerID:    in.CustomerID,
		BookingID:     in.BookingID,
		BankAccountID: in.BankAccountID,
		BankCode:      in.BankCode,
		Method:        in.Method,
		Status:        status,
		AmountCents:   in.AmountCents,
		Currency:      in.Currency,
		Reference:     in.Reference,
		ReceiptURL:    in.ReceiptURL,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.payments.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) MarkPaid(ctx context.Context, tenantID, paymentID uuid.UUID) (*domain.Payment, error) {
	p, err := s.payments.GetByID(ctx, tenantID, paymentID)
	if err != nil {
		return nil, err
	}
	if p.Status == domain.PaymentVoided {
		return nil, fmt.Errorf("%w: payment voided", domain.ErrForbidden)
	}
	p.Status = domain.PaymentPaid
	p.UpdatedAt = s.clock.Now()
	if err := s.payments.Update(ctx, p); err != nil {
		return nil, err
	}
	if p.BookingID != nil {
		if b, err := s.bookings.GetByID(ctx, tenantID, *p.BookingID); err == nil {
			b.OperativeStage = domain.StagePaid
			b.UpdatedAt = p.UpdatedAt
			_ = s.bookings.Update(ctx, b)
		}
	}
	return p, nil
}

func (s *Service) Void(ctx context.Context, tenantID, paymentID uuid.UUID) (*domain.Payment, error) {
	p, err := s.payments.GetByID(ctx, tenantID, paymentID)
	if err != nil {
		return nil, err
	}
	p.Status = domain.PaymentVoided
	p.UpdatedAt = s.clock.Now()
	if err := s.payments.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	return s.payments.GetByID(ctx, tenantID, id)
}

func (s *Service) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Payment, error) {
	return s.payments.ListByCustomer(ctx, tenantID, customerID)
}

func (s *Service) ListBanks(ctx context.Context, tenantID uuid.UUID) ([]domain.BankAccount, error) {
	return s.banks.ListActive(ctx, tenantID)
}

func (s *Service) CreateBank(ctx context.Context, a *domain.BankAccount) error {
	if a.ID == uuid.Nil {
		a.ID = s.ids.New()
	}
	return s.banks.Create(ctx, a)
}
