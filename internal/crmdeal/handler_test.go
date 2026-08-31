package crmdeal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-azhan/api/internal/identity"
	"github.com/google/uuid"
)

func TestProcessPaxValidation(t *testing.T) {
	repo := NewRepository(nil)

	// Pax > 1 should return ErrMaxPaxExceeded without DB connection
	req := DealRequest{
		Pax: 3,
	}
	_, err := repo.Process(context.Background(), 1, 1, uuid.NewString(), req)
	if !errors.Is(err, ErrMaxPaxExceeded) {
		t.Fatalf("expected ErrMaxPaxExceeded, got %v", err)
	}
}

func TestHandlerProcessDealPaxValidation(t *testing.T) {
	repo := NewRepository(nil)
	handler := NewHandler(repo)

	// Case 1: pax = 3 -> rejected with 400 Bad Request
	body, _ := json.Marshal(map[string]any{
		"crm_lead_id": uuid.NewString(),
		"schedule_id": 1,
		"room_type":   "Quad",
		"pax":         3,
		"jamaah": map[string]any{
			"nama_lengkap": "Test Jamaah",
			"no_hp":        "081234567890",
		},
		"commitment_type": "dp",
		"payment_amount":  5000000,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/admin/crm/deals", bytes.NewReader(body))
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.Header.Set("Content-Type", "application/json")

	// Set brand_id in context
	brandID := int64(1)
	ctx := context.WithValue(req.Context(), identity.BrandIDKey, &brandID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ProcessDeal(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedMsg := "Deal CRM saat ini hanya mendukung 1 pax per transaksi. Untuk booking lebih dari 1 orang, gunakan menu Booking di dashboard admin."
	if resp["error"] != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, resp["error"])
	}
}
