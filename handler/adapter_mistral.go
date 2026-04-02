// package handler

// import (
//     "bytes"
//     "encoding/json"
//     "net/http"
// )

// type MistralAdapter struct{
//     OpenAIAdapter 
// }

// func (m *MistralAdapter) PrepareRequest(p, modelName, k, v string) (*http.Request, error) {
//     url := "https://api.mistral.ai/v1/chat/completions"
//     // Use modelName (the parameter) instead of m (the receiver)
//     payload := StreamRequestPayload{
//         Model:    modelName, 
//         Messages: []Message{{Role: "user", Content: p}}, 
//         Stream:   true,
//     }
//     body, _ := json.Marshal(payload)
    
//     req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
//     if err!= nil { return nil, err }
    
//     req.Header.Set("Content-Type", "application/json")
//     req.Header.Set("Authorization", "Bearer "+k)
//     return req, nil
// }


package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type MistralAdapter struct{
    OpenAIAdapter 
}

func (m *MistralAdapter) PrepareRequest(messages []Message, modelName, k, v string) (*http.Request, error) {
    url := "https://api.mistral.ai/v1/chat/completions"
    
    payload := map[string]any{
        "model":    modelName, 
        "messages": messages, 
        "stream":   true,
    }
    
    body, _ := json.Marshal(payload)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil { return nil, err }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+k)
    return req, nil
}