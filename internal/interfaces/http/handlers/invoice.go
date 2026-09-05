package handlers

import (
	"encoding/json"
	"net/http"

	"arca-invoice-proxy/internal/application/invoice"
	"arca-invoice-proxy/internal/interfaces/http/dto"
	"arca-invoice-proxy/internal/interfaces/http/middleware"
)

type InvoiceHandler struct {
	service *invoice.Service
}

func NewInvoiceHandler(service *invoice.Service) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

func (h *InvoiceHandler) IssueInvoice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req dto.IssueInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.IdempotencyKey = idempotencyKey

	input := invoice.IssueInvoiceInput{
		CustomerID:     userID,
		InvoiceType:    req.InvoiceType,
		PointOfSale:    req.PointOfSale,
		Items:          make([]invoice.InvoiceItemInput, len(req.Items)),
		IdempotencyKey: req.IdempotencyKey,
	}

	for i, item := range req.Items {
		input.Items[i] = invoice.InvoiceItemInput{
			Description:    item.Description,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	output, err := h.service.IssueInvoice(r.Context(), input)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := dto.IssueInvoiceResponse{
		InvoiceID:     output.Invoice.ID,
		Status:        output.Invoice.Status.String(),
		InvoiceType:   output.Invoice.Type.String(),
		PointOfSale:   output.Invoice.PointOfSale,
		SubtotalCents: output.Invoice.SubtotalCents,
		TaxCents:      output.Invoice.TaxCents,
		TotalCents:    output.Invoice.TotalCents,
		Items:         make([]dto.InvoiceItemResponse, len(output.Invoice.Items)),
		CreatedAt:     output.Invoice.CreatedAt,
	}

	for i, item := range output.Invoice.Items {
		resp.Items[i] = dto.ToInvoiceItemResponse(item)
	}

	if output.Invoice.ARCAResponse != nil {
		resp.CAE = output.Invoice.ARCAResponse.CAE
		resp.CAEExpiresAt = &output.Invoice.ARCAResponse.CAEExpirationDate
		resp.VoucherNumber = output.Invoice.ARCAResponse.VoucherNumber
		resp.VoucherType = output.Invoice.ARCAResponse.VoucherType
	}

	if output.Invoice.IssuedAt != nil {
		resp.IssuedAt = output.Invoice.IssuedAt
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: message,
		Code:  "error",
	})
}