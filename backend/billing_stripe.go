package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v83"
	checkoutsession "github.com/stripe/stripe-go/v83/checkout/session"
	"github.com/stripe/stripe-go/v83/webhook"
)

type StripeHandler struct {
	cfg *Config
}

func NewStripeHandler(cfg *Config) *StripeHandler {
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}
	return &StripeHandler{cfg: cfg}
}

// POST /billing/create-checkout-session
// Early support: uses STRIPE_PRICE_ID and redirects to AppBaseURL.
func (h *StripeHandler) handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if h.cfg.StripeSecretKey == "" || h.cfg.StripeDefaultPriceID == "" {
		writeJson(w, http.StatusNotImplemented, map[string]string{"error": "stripe not configured"})
		return
	}
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	successURL := fmt.Sprintf("%s/billing/success?session_id={CHECKOUT_SESSION_ID}", h.cfg.AppBaseURL)
	cancelURL := fmt.Sprintf("%s/billing/cancel", h.cfg.AppBaseURL)

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(h.cfg.StripeDefaultPriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJson(w, http.StatusOK, map[string]any{
		"id":  session.ID,
		"url": session.URL,
	})
}

// POST /billing/webhook
func (h *StripeHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if h.cfg.StripeWebhookSecret == "" {
		writeJson(w, http.StatusNotImplemented, map[string]string{"error": "stripe webhook not configured"})
		return
	}
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJson(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "payload too large"})
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, h.cfg.StripeWebhookSecret)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}

	log.Printf("stripe event received: %s", event.Type)
	// Early support: just log and ack.

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"received": true})
}
