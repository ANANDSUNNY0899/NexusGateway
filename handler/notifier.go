package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"log"
	"os"
)

func Notify(message string) {
	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" || chatID == "" { return }

	
	formattedMsg := fmt.Sprintf("[NEXUS PULSE v3.1]\n\n%s", message)
	
	apiURL := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s",
		token,
		chatID,
		url.QueryEscape(formattedMsg),
	)

	go func() {
		resp, err := http.Get(apiURL)
		if err != nil { return }
		defer resp.Body.Close()
		log.Printf("📡 Pulse status: %s", resp.Status)
	}()
}