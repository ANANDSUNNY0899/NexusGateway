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
	// 🚀 THE SECURE WAY: Only fetch from System Environment
	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	// Safety check: Don't crash, just skip if not configured
	if token == "" || chatID == "" {
		log.Println("⚠️  Pulse skipped: Telegram credentials missing in ENV")
		return
	}

	// Professional Header
	formattedMsg := fmt.Sprintf("[NEXUS PULSE v3.1]\n\n%s", message)
	
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s&parse_mode=Markdown",
		token,
		chatID,
		url.QueryEscape(formattedMsg),
	)

	// Async Call: Performance first
	go func() {
		log.Printf("📡 Attempting to send Pulse to Telegram...")
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("🚨 Pulse Connection Error: %v", err)
			return
		}
		defer resp.Body.Close()
		log.Printf("✅ Pulse status: %s", resp.Status)
	}()
}