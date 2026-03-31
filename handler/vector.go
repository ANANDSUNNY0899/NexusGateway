package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"strings"
)

// --- 1. GENERATE DENSE VECTOR (Semantic Meaning) ---

// func GetEmbedding(text string, apiKey string) ([]float32, error) {
// 	url := "https://api.openai.com/v1/embeddings"
// 	payload := map[string]string{"input": text, "model": "text-embedding-3-small"}
// 	body, _ := json.Marshal(payload)
	
// 	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
// 	req.Header.Set("Authorization", "Bearer "+apiKey)
// 	req.Header.Set("Content-Type", "application/json")
	
// 	resp, err := (&http.Client{}).Do(req)
// 	if err != nil { 
// 		log.Printf("❌ [DEBUG] Embedding Network Error: %v", err)
// 		return nil, err 
// 	}
// 	defer resp.Body.Close()
	
//     // Read body to see OpenAI error message
//     respBody, _ := io.ReadAll(resp.Body)
//     if resp.StatusCode != 200 {
//         log.Printf("❌ [DEBUG] OpenAI API Error (%d): %s", resp.StatusCode, string(respBody))
//         return nil, fmt.Errorf("openai error: %s", string(respBody))
//     }

// 	var res struct { 
//         Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"` 
//     }
// 	json.NewDecoder(bytes.NewReader(respBody)).Decode(&res)
	
//     if len(res.Data) > 0 { 
//         return res.Data[0].Embedding, nil 
//     }
// 	return nil, fmt.Errorf("no embedding data returned")
// }

func GetEmbedding(text string, apiKey string) ([]float32, error) {
    url := "https://api.openai.com/v1/embeddings"
    payload := map[string]any{
        "input": text, 
        "model": "text-embedding-3-small",
    }
    body, _ := json.Marshal(payload)
    
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := (&http.Client{}).Do(req)
    if err != nil { 
        log.Printf("❌ [DEBUG] Network Error: %v", err)
        return nil, err 
    }
    defer resp.Body.Close()
    
    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != 200 {
        log.Printf("❌ [DEBUG] OpenAI Error (%d): %s", resp.StatusCode, string(respBody))
        return nil, fmt.Errorf("openai error: %s", string(respBody))
    }

    var res struct { 
        Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"` 
    }
    
    if err := json.Unmarshal(respBody, &res); err != nil {
        log.Printf("❌ [DEBUG] JSON Unmarshal Error: %v", err)
        return nil, err
    }
    
    if len(res.Data) > 0 { 
        emb := res.Data[0].Embedding
        
        // --- 🔍 THE "WHY IS MY SCORE 0.08" CHECKS ---
        log.Printf("📊 [DEBUG] Vector Size: %d", len(emb)) // MUST BE 1536
        if len(emb) > 0 {
            log.Printf("📊 [DEBUG] Sample (first 3): [%.4f, %.4f, %.4f]", emb[0], emb[1], emb[2])
        }
        
        return emb, nil 
    }
    return nil, fmt.Errorf("no embedding data returned")
}

// --- 2. GENERATE SPARSE VECTOR (Keyword Frequency) ---
// Fixed: Return types changed to []uint32 and []float32 to match FNV output and Pinecone expectations
func GenerateSparseVector(text string) ([]uint32, []float32) {
	words := strings.Fields(strings.ToLower(text))
	freqMap := make(map[uint32]float32)

	for _, word := range words {
		h := fnv.New32a()
		h.Write([]byte(word))
		wordID := h.Sum32()
		freqMap[wordID] += 1.0 
	}

	var indices []uint32
	var values []float32
	
	for id, freq := range freqMap {
		indices = append(indices, id)
		values = append(values, freq)
	}
	return indices, values
}

// --- 3. SEARCH PINECONE ---
func SearchPinecone(host, key string, denseVector []float32) (string, float64, error) {
    // FIX 1: Ensure the host doesn't have https:// doubled up
    cleanHost := strings.TrimPrefix(host, "https://")
    url := fmt.Sprintf("https://%s/query", cleanHost)
    
    payload := map[string]any{
        "vector":          denseVector,
        "topK":            1,
        "includeMetadata": true,
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Api-Key", key)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := (&http.Client{}).Do(req)
    if err != nil {
        return "", 0, err
    }
    defer resp.Body.Close()

    // FIX 2: Check for non-200 status codes (Pinecone might be rejecting the API Key or Host)
    if resp.StatusCode != 200 {
        respBody, _ := io.ReadAll(resp.Body)
        log.Printf("❌ [DEBUG] Pinecone Search Error (%d): %s", resp.StatusCode, string(respBody))
        return "", 0, fmt.Errorf("pinecone error: %d", resp.StatusCode)
    }
    
    var res struct { 
        Matches []struct { 
            Score    float64           `json:"score"`
            Metadata map[string]string `json:"metadata"` 
        } `json:"matches"` 
    }
    
    // Use a decoder to parse the result
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        return "", 0, err
    }
    
    if len(res.Matches) > 0 {
        // FIX 3: Defensive check - make sure the "answer" key actually exists in metadata
        answer, ok := res.Matches[0].Metadata["answer"]
        if !ok || answer == "" {
            log.Printf("⚠️ [DEBUG] Match found (Score: %.2f) but 'answer' metadata is missing!", res.Matches[0].Score)
            return "", res.Matches[0].Score, nil
        }
        return answer, res.Matches[0].Score, nil
    }
    
    return "", 0, nil
}


// --- SYNCED: SAVE TO PINECONE HYBRID ---
// Definition now matches the 7 arguments seen in terminal
func SaveToPineconeHybrid(host, key, id string, denseVector []float32, sparseIndices []uint32, sparseValues []float32, answer string) {
	log.Printf("💾 [DEBUG] Attempting Pinecone Hybrid Upsert for ID: %s", id)
	
	url := fmt.Sprintf("https://%s/vectors/upsert", host)
	
	// Prepare payload with conditional sparse values
	vectorData := map[string]any{
		"id":       id,
		"values":   denseVector,
		"metadata": map[string]string{"answer": answer},
	}

	// Only add sparseValues if they aren't empty/nil
	if len(sparseIndices) > 0 {
		vectorData["sparseValues"] = map[string]any{
			"indices": sparseIndices,
			"values":  sparseValues,
		}
	}

	payload := map[string]any{
		"vectors": []map[string]any{vectorData},
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Api-Key", key)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Printf("❌ [DEBUG] Pinecone Error: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("❌ [DEBUG] Pinecone Error Body: %s", string(respBody))
	} else {
		log.Println("✅ [DEBUG] Pinecone Success!")
	}
}

// --- SYNCED: SAVE TO PINECONE (The 5-argument wrapper for chat.go) ---
func SaveToPinecone(host, key, id string, denseVector []float32, answer string) {
	// We call the 7-argument version and pass 'nil' for the sparse parts
	SaveToPineconeHybrid(host, key, id, denseVector, nil, nil, answer)
}

// --- 5. DYNAMIC THRESHOLD ---

func CalculateDynamicThreshold(text string, baseThreshold float64, avgSimilarity float64) float64 {
    textLower := strings.ToLower(text)

    // 1. HIGH PRECISION (Logic/Technical)
    // We want 0.85+ for these because a small change in code is a different answer
    techTerms := []string{"code", "func", "python", "javascript", "go", "sql", "math", "api", "error"}
    for _, term := range techTerms {
        if strings.Contains(textLower, term) {
            return 0.72
        }
    }

    // 2. CONVERSATIONAL/GREETINGS (Low Precision)
    // "Hello" and "Hey" should always hit the cache even with a 0.60 score
    socialTerms := []string{"hello", "hi", "hey", "morning", "thanks", "bye"}
    for _, term := range socialTerms {
        if strings.Contains(textLower, term) {
            return 0.60 
        }
    }

    // 3. DYNAMIC ADJUSTMENT (General Knowledge)
    // If the cache is very "dense" (avgSimilarity is high), we tighten the threshold
    if avgSimilarity > 0.8 {
        return baseThreshold + 0.05
    }
    
    return baseThreshold // Usually 0.68 - 0.70
}