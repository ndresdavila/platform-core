package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrValidation     = errors.New("validation")
	ErrTenantInactive = errors.New("tenant inactive")
	ErrForbidden      = errors.New("forbidden")
)

func (b Booking) CanCustomerCancel(payment *Payment) error {
	if b.Status == BookingCancelled || b.Status == BookingCompleted {
		return ErrForbidden
	}
	if payment != nil && payment.Method == PaymentTransfer &&
		(payment.Status == PaymentReported || payment.Status == PaymentPaid) {
		return ErrForbidden
	}
	return nil
}

func NextOperativeStage(current OperativeStage) (OperativeStage, error) {
	switch current {
	case StageBooked:
		return StageInProgress, nil
	case StageInProgress:
		return StagePaid, nil
	case StagePaid:
		return StageFinished, nil
	default:
		return current, ErrValidation
	}
}
