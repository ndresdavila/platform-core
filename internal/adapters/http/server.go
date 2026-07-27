package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/ndresdavila/platform-core/internal/application/identity"
	"github.com/ndresdavila/platform-core/internal/application/invoice"
	"github.com/ndresdavila/platform-core/internal/application/tenant"
	"github.com/ndresdavila/platform-core/internal/domain"
)

type Server struct {
	Tenants  *tenant.Service
	Identity *identity.Service
	Invoices *invoice.Service
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
			r.Post("/employees/upsert", s.upsertEmployee)

			r.Get("/persons", s.listPersons)
			r.Post("/persons", s.createPerson)
			r.Get("/products", s.listProducts)
			r.Post("/products", s.createProduct)
			r.Get("/invoices", s.listInvoices)
			r.Post("/invoices", s.createInvoice)
			r.Get("/invoices/{invoiceID}", s.getInvoice)
			r.Post("/invoices/{invoiceID}/send-sri", s.sendInvoiceSRI)
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

func (s *Server) listPersons(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	list, err := s.Invoices.ListPersons(r.Context(), tenantID, r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body invoice.CreatePersonInput
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	body.TenantID = tenantID
	p, err := s.Invoices.CreatePerson(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	list, err := s.Invoices.ListProducts(r.Context(), tenantID, r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body invoice.CreateProductInput
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	body.TenantID = tenantID
	p, err := s.Invoices.CreateProduct(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) listInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	list, err := s.Invoices.ListDocuments(r.Context(), tenantID, 50)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "invoiceID")
	if !ok {
		return
	}
	doc, err := s.Invoices.GetDocument(r.Context(), tenantID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) createInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	var body struct {
		DocType              string              `json:"docType"`
		PartyKind            string              `json:"partyKind"`
		PersonID             *uuid.UUID          `json:"personId"`
		PersonName           string              `json:"personName"`
		PersonIdentification string              `json:"personIdentification"`
		Establishment        string              `json:"establishment"`
		EmissionPoint        string              `json:"emissionPoint"`
		IssueDate            string              `json:"issueDate"`
		DueDays              int                 `json:"dueDays"`
		Reference            string              `json:"reference"`
		Seller               string              `json:"seller"`
		Description          string              `json:"description"`
		IsExport             bool                `json:"isExport"`
		SendToSRI            bool                `json:"sendToSri"`
		CreatedByExternalID  string              `json:"createdByExternalId"`
		Lines                []invoice.LineInput `json:"lines"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	var issueDate time.Time
	if body.IssueDate != "" {
		var err error
		issueDate, err = time.Parse("2006-01-02", body.IssueDate)
		if err != nil {
			issueDate, err = time.Parse("02/01/2006", body.IssueDate)
			if err != nil {
				writeErr(w, domain.ErrValidation)
				return
			}
		}
	}
	doc, err := s.Invoices.CreateDocument(r.Context(), invoice.CreateDocumentInput{
		TenantID: tenantID, DocType: body.DocType, PartyKind: body.PartyKind,
		PersonID: body.PersonID, PersonName: body.PersonName, PersonIdentification: body.PersonIdentification,
		Establishment: body.Establishment, EmissionPoint: body.EmissionPoint, IssueDate: issueDate,
		DueDays: body.DueDays, Reference: body.Reference, Seller: body.Seller, Description: body.Description,
		IsExport: body.IsExport, SendToSRI: body.SendToSRI, CreatedByExternalID: body.CreatedByExternalID,
		Lines: body.Lines,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) sendInvoiceSRI(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := tenantAndID(w, r, "invoiceID")
	if !ok {
		return
	}
	doc, err := s.Invoices.SendToSRI(r.Context(), tenantID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
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
