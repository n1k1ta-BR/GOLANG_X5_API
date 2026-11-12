package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/order-service/internal/order"

	"github.com/go-chi/chi/v5"
)

const maxBodyBytes = 1 << 20

// decodeJSON принемает одно json значение
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("body must contain one JSON object")
	}
	return nil
}

func orderID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, order.ErrInvalidOrder
	}
	return id, nil
}

// @Summary Создать заказ
// @Tags orders
// @Accept json
// @Produce json
// @Param input body order.CreateOrderInput true "Позиции заказа"
// @Success 201 {object} order.Order
// @Header 201 {string} Location "Адрес созданного заказа"
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders [post]
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var input order.CreateOrderInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, 400, errorResponse{"invalid JSON body", "INVALID_JSON"})
		return
	}
	value, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/orders/%d", value.ID))
	writeJSON(w, http.StatusCreated, value)
}

// @Summary Получить список заказов
// @Tags orders
// @Produce json
// @Success 200 {array} order.Order
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders [get]
func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	values, err := h.service.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, values)
}

// @Summary Получить заказ по ID
// @Tags orders
// @Produce json
// @Param id path int true "ID заказа" minimum(1)
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id} [get]
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := orderID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	value, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, value)
}

// @Summary Изменить статус заказа
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int true "ID заказа" minimum(1)
// @Param input body order.UpdateStatusInput true "Новый статус"
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id}/status [patch]
func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := orderID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var input order.UpdateStatusInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeJSON(w, 400, errorResponse{"invalid JSON body", "INVALID_JSON"})
		return
	}
	value, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, value)
}

// @Summary Получить историю статусов заказа
// @Tags orders
// @Produce json
// @Param id path int true "ID заказа" minimum(1)
// @Success 200 {array} order.StatusChange
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders/{id}/history [get]
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	id, err := orderID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	values, err := h.service.History(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, values)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	response := errorResponse{"internal server error", "INTERNAL_ERROR"}
	switch {
	case errors.Is(err, order.ErrInvalidOrder):
		status = 400
		response = errorResponse{"invalid order or id", "INVALID_ORDER"}
	case errors.Is(err, order.ErrInvalidStatus):
		status = 400
		response = errorResponse{"invalid status", "INVALID_STATUS"}
	case errors.Is(err, order.ErrOrderNotFound):
		status = 404
		response = errorResponse{"order not found", "ORDER_NOT_FOUND"}
	case errors.Is(err, order.ErrInvalidStatusTransition):
		status = 409
		response = errorResponse{"status transition is not allowed", "INVALID_STATUS_TRANSITION"}
	}
	if status == 500 {
		h.logger.ErrorContext(r.Context(), "request failed", slog.Any("error", err), slog.String("request_id", requestID(r.Context())))
	}
	writeJSON(w, status, response)
}
