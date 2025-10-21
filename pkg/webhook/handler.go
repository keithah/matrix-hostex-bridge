package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"maunium.net/go/mautrix/bridgev2"
)

type WebhookHandler struct {
	br     *bridgev2.Bridge
	server *http.Server
	router *mux.Router
}

type WebhookEvent struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type MessageWebhookData struct {
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	Content        string    `json:"content"`
	SenderType     string    `json:"sender_type"`
	SenderName     string    `json:"sender_name"`
	Timestamp      time.Time `json:"timestamp"`
}

type ConversationWebhookData struct {
	ConversationID string `json:"conversation_id"`
	PropertyID     string `json:"property_id"`
	ReservationID  string `json:"reservation_id"`
	GuestName      string `json:"guest_name"`
	GuestEmail     string `json:"guest_email"`
	Status         string `json:"status"`
}

func NewWebhookHandler(bridge *bridgev2.Bridge, port int) *WebhookHandler {
	router := mux.NewRouter()

	wh := &WebhookHandler{
		br:     bridge,
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
	}

	// Setup routes
	router.HandleFunc("/webhook/hostex", wh.handleHostexWebhook).Methods("POST")
	router.HandleFunc("/health", wh.handleHealth).Methods("GET")

	return wh
}

func (wh *WebhookHandler) Start(ctx context.Context) error {
	wh.br.Log.Info().Str("addr", wh.server.Addr).Msg("Starting webhook server")

	go func() {
		if err := wh.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			wh.br.Log.Error().Err(err).Msg("Webhook server error")
		}
	}()

	return nil
}

func (wh *WebhookHandler) Stop(ctx context.Context) error {
	wh.br.Log.Info().Msg("Stopping webhook server")
	return wh.server.Shutdown(ctx)
}

func (wh *WebhookHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "mautrix-hostex",
		"timestamp": time.Now().Format(time.RFC3339Nano),
	}); err != nil {
		wh.br.Log.Error().Err(err).Msg("Failed to encode health response")
	}
}

func (wh *WebhookHandler) handleHostexWebhook(w http.ResponseWriter, r *http.Request) {
	wh.br.Log.Debug().Str("method", r.Method).Str("path", r.URL.Path).Msg("Received webhook")

	// TODO: Validate webhook signature if Hostex provides one

	var event WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		wh.br.Log.Error().Err(err).Msg("Failed to decode webhook payload")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch event.Type {
	case "message.created":
		err := wh.handleMessageCreated(ctx, event.Data)
		if err != nil {
			wh.br.Log.Error().Err(err).Msg("Failed to handle message created event")
			http.Error(w, "Failed to process message", http.StatusInternalServerError)
			return
		}

	case "conversation.created":
		err := wh.handleConversationCreated(ctx, event.Data)
		if err != nil {
			wh.br.Log.Error().Err(err).Msg("Failed to handle conversation created event")
			http.Error(w, "Failed to process conversation", http.StatusInternalServerError)
			return
		}

	case "reservation.created":
		err := wh.handleReservationCreated(ctx, event.Data)
		if err != nil {
			wh.br.Log.Error().Err(err).Msg("Failed to handle reservation created event")
			http.Error(w, "Failed to process reservation", http.StatusInternalServerError)
			return
		}

	default:
		wh.br.Log.Debug().Str("event_type", event.Type).Msg("Unhandled webhook event type")
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "processed",
	}); err != nil {
		wh.br.Log.Error().Err(err).Msg("Failed to encode webhook response")
	}
}

func (wh *WebhookHandler) handleMessageCreated(ctx context.Context, data interface{}) error {
	msgDataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	var msgData MessageWebhookData
	if err := json.Unmarshal(msgDataBytes, &msgData); err != nil {
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	wh.br.Log.Info().
		Str("conversation_id", msgData.ConversationID).
		Str("sender_type", msgData.SenderType).
		Str("content", msgData.Content).
		Msg("New message received via webhook")

	// TODO: Forward message to appropriate Matrix room
	// This would involve:
	// 1. Finding the portal for the conversation
	// 2. Converting the message format
	// 3. Sending to Matrix

	return nil
}

func (wh *WebhookHandler) handleConversationCreated(ctx context.Context, data interface{}) error {
	convDataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation data: %w", err)
	}

	var convData ConversationWebhookData
	if err := json.Unmarshal(convDataBytes, &convData); err != nil {
		return fmt.Errorf("failed to unmarshal conversation data: %w", err)
	}

	wh.br.Log.Info().
		Str("conversation_id", convData.ConversationID).
		Str("property_id", convData.PropertyID).
		Str("guest_name", convData.GuestName).
		Msg("New conversation created via webhook")

	// TODO: Create new Matrix room for conversation
	// This would involve:
	// 1. Creating a portal for the conversation
	// 2. Setting up the Matrix room
	// 3. Inviting appropriate users

	return nil
}

func (wh *WebhookHandler) handleReservationCreated(ctx context.Context, data interface{}) error {
	wh.br.Log.Info().Interface("data", data).Msg("New reservation created via webhook")

	// TODO: Handle reservation events
	// This might involve creating notifications or updating room topics

	return nil
}
