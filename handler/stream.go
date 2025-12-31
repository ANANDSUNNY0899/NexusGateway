package handler

import (
	"NexusGateway/config"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// We need a specific request struct for streaming (OpenAI format)
type StreamRequestPayload struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"` // <--- THIS IS THE KEY
}

func HandleStreamChat(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()

	// 1. Set Headers for Streaming (Crucial)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 2. Parse User Request
	var userReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if userReq.Model == "" { userReq.Model = "gpt-3.5-turbo" }

	// 3. Prepare Request to OpenAI
	payload := StreamRequestPayload{
		Model: userReq.Model,
		Messages: []Message{
			{Role: "user", Content: userReq.Message},
		},
		Stream: true, // Tell OpenAI to stream
	}
	jsonBody, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	// 4. Execute Request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(w, "data: Error connecting to OpenAI\n\n")
		return
	}
	defer resp.Body.Close()

	// 5. THE PIPELINE (Read from OpenAI -> Write to User)
	reader := bufio.NewReader(resp.Body)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		// Read a line from OpenAI
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break // End of stream
		}

		// OpenAI sends lines like: "data: { ... json ... }"
		// We just forward them directly to the user
		lineStr := string(line)
		
		if strings.HasPrefix(lineStr, "data: ") {
			// Write to our client
			w.Write(line)
			// FLUSH instantly (Don't wait for buffer to fill)
			flusher.Flush()
		}
	}
}