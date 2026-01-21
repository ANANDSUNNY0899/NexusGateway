package handler

import (
	"encoding/json"
	"net/http"
)

func HandleCheckout(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"checkout_url": "https://buy.stripe.com/test_yourlink"})
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}