package handler

import (
	"NexusGateway/config"
	"bufio"
	"bytes"
	"context" // <--- Added this
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type StreamRequestPayload struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()

	// 1. REDIS COUNTING (Fixed)
	// We use redisClient variable to avoid conflict later
	redisClient := GetClient()
	if redisClient != nil {
		redisClient.Incr(context.Background(), "stats:total_requests")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	payload := StreamRequestPayload{
		Model: userReq.Model,
		Messages: []Message{
			{Role: "user", Content: userReq.Message},
		},
		Stream: true,
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	// 2. HTTP CLIENT (Renamed to avoid conflict)
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(w, "data: Error connecting to OpenAI\n\n")
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break 
		}
		lineStr := string(line)
		
		if strings.HasPrefix(lineStr, "data: ") {
			w.Write(line)
			flusher.Flush()
		}
	}
}