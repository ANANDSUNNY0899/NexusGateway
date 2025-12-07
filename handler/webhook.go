package handler

import (
	"NexusGateway/config"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// 1. Verify the Signature (Security Check)
	// This ensures the request is actually from Stripe
	sigHeader := r.Header.Get("Stripe-Signature")

	log.Printf("DEBUG: Config Key: %s", cfg.StripeWebhookSecret)

	//event, err := webhook.ConstructEvent(payload, sigHeader, cfg.StripeWebhookSecret)

	// Use Options to ignore the version mismatch error
	event, err := webhook.ConstructEventWithOptions(
		payload,
		sigHeader,
		cfg.StripeWebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)

	if err != nil {
		log.Printf("⚠️ Signature Error: %v", err) // Print the ACTUAL error
		w.WriteHeader(http.StatusBadRequest)
		return
    }

	// 2. Handle the "Checkout Completed" Event
	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 3. Get the User API Key from Metadata
		userKey := session.Metadata["user_api_key"]
		if userKey != "" {
			// 4. Upgrade the User in Database!
			UpgradeUser(userKey)
		}
	}

	w.WriteHeader(http.StatusOK)
}