package handler

import (
	"NexusGateway/config"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"io"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

func HandleCheckout(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	stripe.Key = cfg.StripeSecretKey

	// 1. Get Nexus API Key from Header
	authHeader := r.Header.Get("Authorization")
	apiKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	if apiKey == "" {
		respondWithError(w, "Unauthorized: Key missing", http.StatusUnauthorized)
		return
	}

	// 2. Setup Stripe Session
	// Note: Replace 'price_...' with your actual Stripe Price ID
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String("https://nexus-gateway.org/dashboard?success=true"),
		CancelURL:  stripe.String("https://nexus-gateway.org/dashboard?canceled=true"),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID: stripe.String(apiKey), // 🚀 CRITICAL: Link payment to Nexus User
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String("price_prod_TYo4Y37GMHO8DW"),
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("❌ STRIPE ERROR: %v", err) // 🚀 This will show the exact reason in terminal
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Stripe Session Creation Failed",
			"details": err.Error(), // 🚀 Tell frontend what went wrong
		})
		return
	}



	// 3. Send URL to Frontend
	json.NewEncoder(w).Encode(map[string]string{
		"checkout_url": s.URL,
	})
}

// HandleWebhook captures the successful payment signal
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Verify Webhook Signature (Safety First)
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), cfg.StripeWebhookSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err == nil {
			apiKey := session.ClientReferenceID
			log.Printf("💰 [PAYMENT] Upgrading User Token: %s", apiKey)
			UpgradeUser(apiKey) // Call DB Upgrade
		}
	}

	w.WriteHeader(http.StatusOK)
}