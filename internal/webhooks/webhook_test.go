package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lunoxd/cobalt/internal/jobs"
)

func TestVerifyHMACSHA256(t *testing.T) {
	secret := "test_secret_123"
	payload := []byte(`{"event":"payment.captured","amount":5000}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	if !VerifyHMACSHA256(payload, validSig, secret) {
		t.Errorf("expected valid signature to pass verification")
	}

	if VerifyHMACSHA256(payload, "invalid_signature", secret) {
		t.Errorf("expected invalid signature to fail verification")
	}

	if VerifyHMACSHA256(payload, validSig, "wrong_secret") {
		t.Errorf("expected wrong secret to fail verification")
	}
}

func TestWebhookHandler_ReceiveAndEnqueue(t *testing.T) {
	pool := jobs.NewPool(1, 10)
	pool.Start()
	defer pool.Stop()

	handler := NewHandler(nil, pool, "")
	routes := handler.Routes()

	payload := `{"id":"evt_12345","event":"order_paid","amount":99}`
	req := httptest.NewRequest(http.MethodPost, "/razorpay", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["status"] != "accepted" {
		t.Errorf("expected status accepted, got %v", res["status"])
	}
	if res["event_id"] != "evt_12345" {
		t.Errorf("expected event_id evt_12345, got %v", res["event_id"])
	}
}
