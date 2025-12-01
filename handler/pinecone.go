package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PineconeVector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpsertRequest struct {
	Vectors []PineconeVector `json:"vectors"`
}

type QueryRequest struct {
	Vector          []float32 `json:"vector"`
	TopK            int       `json:"topK"`
	IncludeMetadata bool      `json:"includeMetadata"`
}

type QueryResponse struct {
	Matches []struct {
		Score    float64                `json:"score"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"matches"`
}

// SaveToPinecone stores the vector and the answer
func SaveToPinecone(host, apiKey, id string, vector []float32, answer string) error {
	url := fmt.Sprintf("https://%s/vectors/upsert", host)

	payload := UpsertRequest{
		Vectors: []PineconeVector{
			{
				ID:     id,
				Values: vector,
				Metadata: map[string]interface{}{
					"response": answer, // We store the text answer inside Pinecone
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinecone upsert failed: %s", string(b))
	}
	return nil
}

// SearchPinecone looks for similar questions
func SearchPinecone(host, apiKey string, vector []float32) (string, float64, error) {
	url := fmt.Sprintf("https://%s/query", host)

	payload := QueryRequest{
		Vector:          vector,
		TopK:            1,
		IncludeMetadata: true,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result QueryResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Matches) > 0 {
		match := result.Matches[0]
		// Retrieve the stored text answer
		if answer, ok := match.Metadata["response"].(string); ok {
			return answer, match.Score, nil
		}
	}

	return "", 0, nil
}