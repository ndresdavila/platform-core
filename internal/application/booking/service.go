package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/domain"
	"github.com/ndresdavila/platform-core/internal/domain/ports"
)

type Service struct {
	tenants   ports.TenantRepository
	customers ports.CustomerRepository
	catalog   ports.CatalogRepository
	bookings  ports.BookingRepository
	payments  ports.PaymentRepository
	clock     ports.Clock
	ids       ports.IDGen
}

func NewService(
	tenants ports.TenantRepository,
	customers ports.CustomerRepository,
	catalog ports.CatalogRepository,
	bookings ports.BookingRepository,
	payments ports.PaymentRepository,
	clock ports.Clock,
	ids ports.IDGen,
) *Service {
	return &Service{
		tenants: tenants, customers: customers, catalog: catalog,
		bookings: bookings, payments: payments, clock: clock, ids: ids,
	}
}

type CreateInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	ServiceID  uuid.UUID
	DesignID   *uuid.UUID
	StartsAt   time.Time
	Notes      string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Booking, error) {
	if err := s.requireActiveTenant(ctx, in.TenantID); err != nil {
		return nil, err
	}
	if in.StartsAt.Before(s.clock.Now()) {
		return nil, fmt.Errorf("%w: starts_at must be in the future", domain.ErrValidation)
	}
	if _, err := s.customers.GetByID(ctx, in.TenantID, in.CustomerID); err != nil {
		return nil, err
	}
	svc, err := s.catalog.GetService(ctx, in.TenantID, in.ServiceID)
	if err != nil {
		return nil, err
	}
	if !svc.Active {
		return nil, fmt.Errorf("%w: service inactive", domain.ErrValidation)
	}

	now := s.clock.Now()
	b := &domain.Booking{
		ID:             s.ids.New(),
		TenantID:       in.TenantID,
		CustomerID:     in.CustomerID,
		ServiceID:      in.ServiceID,
		DesignID:       in.DesignID,
		StartsAt:       in.StartsAt,
		Status:         domain.BookingPending,
		OperativeStage: domain.StageBooked,
		Notes:          in.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.bookings.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Booking, error) {
	if err := s.requireActiveTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.bookings.GetByID(ctx, tenantID, id)
}

func (s *Service) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Booking, error) {
	if err := s.requireActiveTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.bookings.ListByCustomer(ctx, tenantID, customerID)
}

func (s *Service) ListByRange(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.Booking, error) {
	if err := s.requireActiveTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.bookings.ListByRange(ctx, tenantID, from, to)
}

func (s *Service) CancelByCustomer(ctx context.Context, tenantID, bookingID uuid.UUID) (*domain.Booking, error) {
	if err := s.requireActiveTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	b, err := s.bookings.GetByID(ctx, tenantID, bookingID)
	if err != nil {
		return nil, err
	}
	pay, _ := s.payments.GetLatestByBooking(ctx, tenantID, bookingID)
	if err := b.CanCustomerCancel(pay); err != nil {
		return nil, err
	}
	b.Status = domain.BookingCancelled
	b.OperativeStage = domain.StageCancelled
	b.UpdatedAt = s.clock.Now()
	if err := s.bookings.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) CancelByStaff(ctx context.Context, tenantID, bookingID uuid.UUID) (*domain.Booking, error) {
	b, err := s.bookings.GetByID(ctx, tenantID, bookingID)
	if err != nil {
		return nil, err
	}
	b.Status = domain.BookingCancelled
	b.OperativeStage = domain.StageCancelled
	b.UpdatedAt = s.clock.Now()
	if err := s.bookings.Update(ctx, b); err != nil {
		return nil, err
	}
	if pay, err := s.payments.GetLatestByBooking(ctx, tenantID, bookingID); err == nil &&
		pay.Status != domain.PaymentPaid && pay.Status != domain.PaymentVoided {
		pay.Status = domain.PaymentVoided
		pay.UpdatedAt = b.UpdatedAt
		_ = s.payments.Update(ctx, pay)
	}
	return b, nil
}

func (s *Service) Advance(ctx context.Context, tenantID, bookingID uuid.UUID, to *domain.OperativeStage) (*domain.Booking, error) {
	b, err := s.bookings.GetByID(ctx, tenantID, bookingID)
	if err != nil {
		return nil, err
	}
	var next domain.OperativeStage
	if to != nil {
		next = *to
	} else {
		next, err = domain.NextOperativeStage(b.OperativeStage)
		if err != nil {
			return nil, err
		}
	}
	b.OperativeStage = next
	switch next {
	case domain.StageInProgress:
		b.Status = domain.BookingInProgress
	case domain.StagePaid, domain.StageFinished:
		b.Status = domain.BookingCompleted
	case domain.StageCancelled:
		b.Status = domain.BookingCancelled
	}
	b.UpdatedAt = s.clock.Now()
	if err := s.bookings.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) requireActiveTenant(ctx context.Context, tenantID uuid.UUID) error {
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if !t.Active {
		return domain.ErrTenantInactive
	}
	return nil
}
