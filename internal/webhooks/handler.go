package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/database"
	"github.com/lunoxd/cobalt/internal/jobs"
)

// Handler processes incoming webhook notifications.
type Handler struct {
	db     *database.DB
	pool   *jobs.Pool
	secret string
}

// NewHandler creates a new webhook Handler.
func NewHandler(db *database.DB, pool *jobs.Pool, secret string) *Handler {
	return &Handler{
		db:     db,
		pool:   pool,
		secret: secret,
	}
}

// Routes sets up webhook endpoints.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/{provider}", h.HandleWebhook)
	return r
}

// HandleWebhook verifies, logs, deduplicates, and enqueues webhook events.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read webhook body", "provider", provider, "err", err)
		http.Error(w, `{"error":"failed to read payload"}`, http.StatusBadRequest)
		return
	}

	// 1. Signature Verification
	var sigHeader string
	switch provider {
	case "razorpay":
		sigHeader = r.Header.Get("X-Razorpay-Signature")
	default:
		sigHeader = r.Header.Get("X-Hub-Signature-256")
		if sigHeader == "" {
			sigHeader = r.Header.Get("X-Signature")
		}
	}

	if h.secret != "" && sigHeader != "" {
		if !VerifyHMACSHA256(body, sigHeader, h.secret) {
			slog.Warn("webhook signature verification failed", "provider", provider)
			http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
			return
		}
	}

	// 2. Parse payload
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("invalid webhook json", "provider", provider, "err", err)
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}

	// 3. Extract event ID for idempotency
	eventID := extractEventID(r, payload)
	eventType, _ := payload["event"].(string)
	if eventType == "" {
		eventType = "unknown"
	}

	// 4. Idempotency storage check
	if h.db != nil && h.db.Pool != nil {
		query := `
			INSERT INTO webhook_events (event_id, provider, event_type, payload, status)
			VALUES ($1, $2, $3, $4, 'received')
			ON CONFLICT (event_id) DO NOTHING
			RETURNING id;
		`
		var insertedID uuid.UUID
		err = h.db.Pool.QueryRow(r.Context(), query, eventID, provider, eventType, body).Scan(&insertedID)
		if err != nil {
			// If already exists, return 200 to acknowledge receipt without re-processing
			slog.Info("Duplicate webhook event skipped", "provider", provider, "event_id", eventID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":   "duplicate_ignored",
				"event_id": eventID,
			})
			return
		}
	}

	slog.Info("Webhook received",
		"provider", provider,
		"event_id", eventID,
		"event_type", eventType,
	)

	// 5. Enqueue background job for asynchronous processing
	if h.pool != nil {
		_, _ = h.pool.Enqueue("webhook_process", map[string]any{
			"provider":   provider,
			"event_id":   eventID,
			"event_type": eventType,
			"payload":    payload,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "accepted",
		"event_id": eventID,
	})
}

func extractEventID(r *http.Request, payload map[string]any) string {
	if headerID := r.Header.Get("X-Event-ID"); headerID != "" {
		return headerID
	}
	if id, ok := payload["id"].(string); ok && id != "" {
		return id
	}
	if eventID, ok := payload["event_id"].(string); ok && eventID != "" {
		return eventID
	}
	return "evt_" + uuid.New().String()
}

// RegisterWebhookJobHandler registers the job processing logic in the background worker pool.
func RegisterWebhookJobHandler(pool *jobs.Pool, db *database.DB) {
	pool.Register("webhook_process", func(ctx context.Context, job *jobs.Job) error {
		provider, _ := job.Payload["provider"].(string)
		eventID, _ := job.Payload["event_id"].(string)
		eventType, _ := job.Payload["event_type"].(string)

		slog.Info("Processing webhook event in background",
			"provider", provider,
			"event_id", eventID,
			"event_type", eventType,
		)

		// Mark processed in database if DB available
		if db != nil && db.Pool != nil && eventID != "" {
			query := `UPDATE webhook_events SET status = 'processed', processed_at = now() WHERE event_id = $1;`
			_, _ = db.Pool.Exec(ctx, query, eventID)
		}

		return nil
	})
}
