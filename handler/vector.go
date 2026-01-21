package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetEmbedding(text string, apiKey string) ([]float32, error) {
	url := "https://api.openai.com/v1/embeddings"
	payload := map[string]string{"input": text, "model": "text-embedding-3-small"}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var res struct { Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"` }
	json.NewDecoder(resp.Body).Decode(&res)
	if len(res.Data) > 0 { return res.Data[0].Embedding, nil }
	return nil, fmt.Errorf("fail")
}

func SearchPinecone(host, key string, vector []float32) (string, float64, error) {
	url := fmt.Sprintf("https://%s/query", host)
	payload := map[string]any{"vector": vector, "topK": 1, "includeMetadata": true}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Api-Key", key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil { return "", 0, err }
	defer resp.Body.Close()
	var res struct { Matches []struct { Score float64 `json:"score"`; Metadata map[string]string `json:"metadata"` } `json:"matches"` }
	json.NewDecoder(resp.Body).Decode(&res)
	if len(res.Matches) > 0 { return res.Matches[0].Metadata["answer"], res.Matches[0].Score, nil }
	return "", 0, nil
}

func SaveToPinecone(host, key, id string, vector []float32, answer string) {
	url := fmt.Sprintf("https://%s/vectors/upsert", host)
	payload := map[string]any{"vectors": []map[string]any{{"id": id, "values": vector, "metadata": map[string]string{"answer": answer}}}}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Api-Key", key)
	req.Header.Set("Content-Type", "application/json")
	(&http.Client{}).Do(req)
}