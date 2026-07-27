package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrValidation     = errors.New("validation")
	ErrTenantInactive = errors.New("tenant inactive")
	ErrForbidden      = errors.New("forbidden")
)
