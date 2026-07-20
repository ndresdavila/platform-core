package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/application/booking"
	"github.com/ndresdavila/platform-core/internal/application/catalog"
	"github.com/ndresdavila/platform-core/internal/application/design"
	"github.com/ndresdavila/platform-core/internal/application/identity"
	"github.com/ndresdavila/platform-core/internal/application/payment"
	"github.com/ndresdavila/platform-core/internal/application/tenant"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type Server struct {
	Tenants  *tenant.Service
	Identity *identity.Service
	Bookings *booking.Service
	Payments *payment.Service
	Catalog  *catalog.Service
	Designs  *design.Service
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/tenants", s.createTenant)
		r.Get("/tenants/by-slug/{slug}", s.getTenantBySlug)

		r.Route("/tenants/{tenantID}", func(r chi.Router) {
			r.Get("/settings", s.getSettings)
			r.Put("/settings", s.putSettings)

			r.Post("/customers/upsert", s.upsertCustomer)
			r.Get("/customers/by-external/{externalId}", s.getCustomerByExternal)
			r.Get("/customers/{id}", s.getCustomer)
			r.Patch("/customers/{id}/profile", s.updateCustomerProfile)
			r.Post("/employees/upsert", s.upsertEmployee)

			r.Get("/catalog/services", s.listServices)
			r.Post("/catalog/services", s.createService)

			r.Post("/bookings", s.createBooking)
			r.Get("/bookings", s.listBookings)
			r.Get("/bookings/{bookingID}", s.getBooking)
			r.Post("/bookings/{bookingID}/cancel-customer", s.cancelBookingByCustomer)
			r.Post("/bookings/{bookingID}/cancel-staff", s.cancelBookingByStaff)
			r.Post("/bookings/{bookingID}/advance", s.advanceBooking)

			r.Post("/payments", s.registerPayment)
			r.Get("/payments", s.listPayments)
			r.Get("/payments/{paymentID}", s.getPayment)
			r.Post("/payments/{paymentID}/mark-paid", s.markPaid)
			r.Post("/payments/{paymentID}/void", s.voidPayment)

			r.Get("/bank-accounts", s.listBanks)
			r.Post("/bank-accounts", s.createBank)

			r.Post("/designs", s.createDesign)
			r.Get("/designs/quota", s.designQuota)
			r.Get("/designs", s.listDesigns)
			r.Get("/designs/{id}", s.getDesign)
			r.Post("/designs/{id}/complete", s.completeDesign)
			r.Delete("/designs/{id}", s.deleteDesign)
		})
	})
	return r
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.Tenants.Create(r.Context(), tenant.CreateInput{Slug: body.Slug, Name: body.Name})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) getTenantBySlug(w http.ResponseWriter, r *http.Request) {
	t, err := s.Tenants.GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	st, err := s.Identity.Settings(r.Context(), tenantID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var st domain.TenantSettings
	if err := decodeJSON(r, &st); err != nil {
		writeErr(w, err)
		return
	}
	st.TenantID = tenantID
	if err := s.Identity.UpsertSettings(r.Context(), &st); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, &st)
}

func (s *Server) upsertCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		ExternalID string  `json:"externalId"`
		Email      string  `json:"email"`
		FirstName  string  `json:"firstName"`
		LastName   string  `json:"lastName"`
		Phone      string  `json:"phone"`
		NationalID *string `json:"nationalId"`
		Role       string  `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	c, err := s.Identity.UpsertCustomer(r.Context(), identity.UpsertCustomerInput{
		TenantID: tenantID, ExternalID: body.ExternalID, Email: body.Email, FirstName: body.FirstName,
		LastName: body.LastName, Phone: body.Phone, NationalID: body.NationalID, Role: body.Role,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) getCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.Identity.GetCustomer(r.Context(), tenantID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) getCustomerByExternal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	c, err := s.Identity.GetCustomerByExternal(r.Context(), tenantID, chi.URLParam(r, "externalId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) updateCustomerProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		NationalID string `json:"nationalId"`
		FirstName  string `json:"firstName"`
		LastName   string `json:"lastName"`
		Phone      string `json:"phone"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	c, err := s.Identity.UpdateCustomerProfile(r.Context(), tenantID, id, body.NationalID, body.FirstName, body.LastName, body.Phone)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) upsertEmployee(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		ExternalID string `json:"externalId"`
		Email      string `json:"email"`
		FirstName  string `json:"firstName"`
		LastName   string `json:"lastName"`
		Role       string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	e, err := s.Identity.UpsertEmployee(r.Context(), identity.UpsertEmployeeInput{
		TenantID: tenantID, ExternalID: body.ExternalID, Email: body.Email, FirstName: body.FirstName, LastName: body.LastName, Role: body.Role,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	items, err := s.Catalog.ListServices(r.Context(), tenantID, r.URL.Query().Get("active") != "false")
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": items})
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body catalog.CreateServiceInput
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	body.TenantID = tenantID
	svc, err := s.Catalog.CreateService(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		CustomerID string    `json:"customerId"`
		ServiceID  string    `json:"serviceId"`
		DesignID   *string   `json:"designId"`
		StartsAt   time.Time `json:"startsAt"`
		Notes      string    `json:"notes"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	customerID, err := uuid.Parse(body.CustomerID)
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	serviceID, err := uuid.Parse(body.ServiceID)
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	var designID *uuid.UUID
	if body.DesignID != nil && *body.DesignID != "" {
		id, err := uuid.Parse(*body.DesignID)
		if err != nil {
			writeErr(w, domain.ErrValidation)
			return
		}
		designID = &id
	}
	b, err := s.Bookings.Create(r.Context(), booking.CreateInput{
		TenantID: tenantID, CustomerID: customerID, ServiceID: serviceID,
		DesignID: designID, StartsAt: body.StartsAt, Notes: body.Notes,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) listBookings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	if customerID := q.Get("customerId"); customerID != "" {
		cid, err := uuid.Parse(customerID)
		if err != nil {
			writeErr(w, domain.ErrValidation)
			return
		}
		items, err := s.Bookings.ListByCustomer(r.Context(), tenantID, cid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"bookings": items})
		return
	}
	from, err1 := time.Parse(time.RFC3339, q.Get("from"))
	to, err2 := time.Parse(time.RFC3339, q.Get("to"))
	if err1 != nil || err2 != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	items, err := s.Bookings.ListByRange(r.Context(), tenantID, from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookings": items})
}

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request) {
	tenantID, bookingID, ok := tenantAndID(w, r, "bookingID")
	if !ok {
		return
	}
	b, err := s.Bookings.Get(r.Context(), tenantID, bookingID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) cancelBookingByCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, bookingID, ok := tenantAndID(w, r, "bookingID")
	if !ok {
		return
	}
	b, err := s.Bookings.CancelByCustomer(r.Context(), tenantID, bookingID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) cancelBookingByStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, bookingID, ok := tenantAndID(w, r, "bookingID")
	if !ok {
		return
	}
	b, err := s.Bookings.CancelByStaff(r.Context(), tenantID, bookingID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) advanceBooking(w http.ResponseWriter, r *http.Request) {
	tenantID, bookingID, ok := tenantAndID(w, r, "bookingID")
	if !ok {
		return
	}
	var body struct {
		To *domain.OperativeStage `json:"to"`
	}
	if err := decodeJSON(r, &body); err != nil && !errors.Is(err, io.EOF) {
		writeErr(w, err)
		return
	}
	b, err := s.Bookings.Advance(r.Context(), tenantID, bookingID, body.To)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) registerPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		CustomerID    string  `json:"customerId"`
		BookingID     *string `json:"bookingId"`
		BankAccountID *string `json:"bankAccountId"`
		BankCode      string  `json:"bankCode"`
		Method        string  `json:"method"`
		AmountCents   int     `json:"amountCents"`
		Currency      string  `json:"currency"`
		Reference     string  `json:"reference"`
		ReceiptURL    string  `json:"receiptUrl"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	customerID, err := uuid.Parse(body.CustomerID)
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	var bookingID *uuid.UUID
	if body.BookingID != nil && *body.BookingID != "" {
		id, err := uuid.Parse(*body.BookingID)
		if err != nil {
			writeErr(w, domain.ErrValidation)
			return
		}
		bookingID = &id
	}
	var bankAccountID *uuid.UUID
	if body.BankAccountID != nil && *body.BankAccountID != "" {
		id, err := uuid.Parse(*body.BankAccountID)
		if err != nil {
			writeErr(w, domain.ErrValidation)
			return
		}
		bankAccountID = &id
	}
	p, err := s.Payments.Register(r.Context(), payment.RegisterInput{
		TenantID: tenantID, CustomerID: customerID, BookingID: bookingID, BankAccountID: bankAccountID, BankCode: body.BankCode,
		Method: domain.PaymentMethod(body.Method), AmountCents: body.AmountCents,
		Currency: body.Currency, Reference: body.Reference, ReceiptURL: body.ReceiptURL,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(r.URL.Query().Get("customerId"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	items, err := s.Payments.ListByCustomer(r.Context(), tenantID, customerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": items})
}

func (s *Server) getPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, paymentID, ok := tenantAndID(w, r, "paymentID")
	if !ok {
		return
	}
	p, err := s.Payments.Get(r.Context(), tenantID, paymentID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) markPaid(w http.ResponseWriter, r *http.Request) {
	tenantID, paymentID, ok := tenantAndID(w, r, "paymentID")
	if !ok {
		return
	}
	p, err := s.Payments.MarkPaid(r.Context(), tenantID, paymentID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) voidPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, paymentID, ok := tenantAndID(w, r, "paymentID")
	if !ok {
		return
	}
	p, err := s.Payments.Void(r.Context(), tenantID, paymentID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listBanks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	items, err := s.Payments.ListBanks(r.Context(), tenantID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bankAccounts": items})
}

func (s *Server) createBank(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var account domain.BankAccount
	if err := decodeJSON(r, &account); err != nil {
		writeErr(w, err)
		return
	}
	account.TenantID = tenantID
	if err := s.Payments.CreateBank(r.Context(), &account); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, &account)
}

func (s *Server) createDesign(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		CustomerID string `json:"customerId"`
		Prompt     string `json:"prompt"`
		PhotoURL   string `json:"photoUrl"`
		ModelUsed  string `json:"modelUsed"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	customerID, err := uuid.Parse(body.CustomerID)
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return
	}
	d, err := s.Designs.Create(r.Context(), design.CreateInput{
		TenantID: tenantID, CustomerID: customerID, Prompt: body.Prompt, PhotoURL: body.PhotoURL, ModelUsed: body.ModelUsed,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) listDesigns(w http.ResponseWriter, r *http.Request) {
	tenantID, customerID, ok := tenantAndQueryID(w, r, "customerId")
	if !ok {
		return
	}
	items, err := s.Designs.List(r.Context(), tenantID, customerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"designs": items})
}

func (s *Server) getDesign(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "id")
	if !ok {
		return
	}
	d, err := s.Designs.Get(r.Context(), tenantID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) completeDesign(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		ResultURL    string `json:"resultUrl"`
		Failed       bool   `json:"failed"`
		ErrorMessage string `json:"errorMessage"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	d, err := s.Designs.Complete(r.Context(), tenantID, id, body.ResultURL, body.Failed, body.ErrorMessage)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) deleteDesign(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "id")
	if !ok {
		return
	}
	if err := s.Designs.Delete(r.Context(), tenantID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) designQuota(w http.ResponseWriter, r *http.Request) {
	tenantID, customerID, ok := tenantAndQueryID(w, r, "customerId")
	if !ok {
		return
	}
	used, limit, totalUsed, totalMax, err := s.Designs.Quota(r.Context(), tenantID, customerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"used": used, "limit": limit, "totalUsed": totalUsed, "totalMax": totalMax,
	})
}

func tenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "tenantID"))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return uuid.Nil, false
	}
	return id, true
}

func tenantAndID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, uuid.UUID, bool) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}

func tenantAndQueryID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, uuid.UUID, bool) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(r.URL.Query().Get(name))
	if err != nil {
		writeErr(w, domain.ErrValidation)
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, domain.ErrValidation):
		status, code = http.StatusBadRequest, "VALIDATION"
	case errors.Is(err, domain.ErrForbidden):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, domain.ErrTenantInactive):
		status, code = http.StatusForbidden, "TENANT_INACTIVE"
	}
	writeJSON(w, status, map[string]string{"error": err.Error(), "code": code})
}
