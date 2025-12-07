package handler

import (
	"NexusGateway/config"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
)

type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

func HandleCheckout(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()
	stripe.Key = cfg.StripeSecretKey

	authHeader := r.Header.Get("Authorization")
	userAPIKey := strings.TrimPrefix(authHeader, "Bearer ")
	userAPIKey = strings.TrimSpace(userAPIKey)

	if userAPIKey == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Nexus Gateway Pro Plan"),
					},
					UnitAmount: stripe.Int64(1000), // $10.00
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		Metadata: map[string]string{
			"user_api_key": userAPIKey,
		},
		// FIXED TYPOS BELOW:
		SuccessURL: stripe.String("https://nexusgateway.onrender.com/success"),
		CancelURL:  stripe.String("https://nexusgateway.onrender.com/cancel"),
	}

	sess, err := session.New(params)
	if err != nil {
		http.Error(w, "Stripe Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CheckoutResponse{CheckoutURL: sess.URL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}