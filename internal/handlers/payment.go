package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"npci-upi/internal/services"
	"npci-upi/internal/types"
)

type PaymentHandler struct {
	PaymentSvc        *services.PaymentService
	ReconciliationSvc *services.ReconciliationService
}

func NewPaymentHandler(ps *services.PaymentService, rs *services.ReconciliationService) *PaymentHandler {
	return &PaymentHandler{
		PaymentSvc:        ps,
		ReconciliationSvc: rs,
	}
}

func (h *PaymentHandler) WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (h *PaymentHandler) WriteError(w http.ResponseWriter, err services.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	idempKey := r.Header.Get("Idempotency-Key")
	correlID := r.Header.Get("X-Correlation-ID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_REQUEST", "cannot read body"))
		return
	}

	var req types.PaymentCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_JSON", err.Error()))
		return
	}

	statusCode, resp, appErr := h.PaymentSvc.CreatePayment(r.Context(), req, idempKey, correlID)
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, statusCode, resp)
}

func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")

	resp, appErr := h.PaymentSvc.GetPaymentStatus(r.Context(), paymentID)
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, 200, resp)
}

func (h *PaymentHandler) ConfirmPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	correlID := r.Header.Get("X-Correlation-ID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_REQUEST", "cannot read body"))
		return
	}

	var req types.ConfirmPaymentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_JSON", err.Error()))
		return
	}

	resp, appErr := h.PaymentSvc.ConfirmPayment(r.Context(), paymentID, correlID)
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, 200, resp)
}

func (h *PaymentHandler) CancelPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	correlID := r.Header.Get("X-Correlation-ID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_REQUEST", "cannot read body"))
		return
	}

	var req types.CancelPaymentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_JSON", err.Error()))
		return
	}

	resp, appErr := h.PaymentSvc.CancelPayment(r.Context(), paymentID, correlID)
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, 200, resp)
}

func (h *PaymentHandler) ManualReversal(w http.ResponseWriter, r *http.Request) {
	correlID := r.Header.Get("X-Correlation-ID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_REQUEST", "cannot read body"))
		return
	}

	var req types.ManualReversalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.WriteError(w, services.NewAppError(400, "INVALID_JSON", err.Error()))
		return
	}

	req2 := types.ReversalRequest{
		OriginalTransactionID: req.OriginalTransactionID,
		Reason:                req.Reason,
	}
	resp, appErr := h.PaymentSvc.ManualReversal(r.Context(), req2, correlID)
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, 201, resp)
}

func (h *PaymentHandler) GetAccountLedger(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 50
	if offsetStr != "" {
		json.Unmarshal([]byte(offsetStr), &offset)
	}
	if limitStr != "" {
		json.Unmarshal([]byte(limitStr), &limit)
	}

	// For now, return a placeholder response
	h.WriteJSON(w, 200, map[string]any{
		"account_id": accountID,
		"entries":    []interface{}{},
		"total":      "0",
	})
}

func (h *PaymentHandler) RunReconciliation(w http.ResponseWriter, r *http.Request) {
	resp, appErr := h.ReconciliationSvc.Run(r.Context())
	if appErr != nil {
		h.WriteError(w, appErr.(services.AppError))
		return
	}

	h.WriteJSON(w, 202, resp)
}

func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func RegisterRoutes(r *chi.Mux, ph *PaymentHandler) {
	r.Get("/health", ph.HealthCheck)
	r.Post("/api/v1/payments", ph.CreatePayment)
	r.Get("/api/v1/payments/{id}", ph.GetPayment)
	r.Post("/api/v1/payments/{id}/confirm", ph.ConfirmPayment)
	r.Post("/api/v1/payments/{id}/cancel", ph.CancelPayment)
	r.Post("/api/v1/reversals", ph.ManualReversal)
	r.Post("/api/v1/reconciliation/run", ph.RunReconciliation)
	r.Get("/api/v1/accounts/{id}/ledger", ph.GetAccountLedger)
}

