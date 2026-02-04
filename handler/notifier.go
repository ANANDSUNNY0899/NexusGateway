package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"log"
	"os"
)

// Notify sends an encrypted signal to the Founder's Telegram
func Notify(message string) {
	// 1. Pehle Environment Variables check karega (Production)
	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	// 2. Agar ENV mein nahi mila, toh aapki di hui keys use karega (Local Testing)
	if token == "" {
		token = "8313635734:AAHklryz4I3yTtqq_IGvn_WTKVWSdJHTyKc"
	}
	if chatID == "" {
		chatID = "5785297510"
	}

	// Safety check
	if token == "" || chatID == "" {
		log.Println("⚠️  Pulse skipped: No Telegram credentials")
		return
	}

	// Format and Escape the message for URL
	msg := url.QueryEscape("🛡️ *[NEXUS PULSE]*\n\n" + message)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=Markdown", token, chatID, msg)

	// Async Execution
	go func() {
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("🚨 Pulse Failed: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}