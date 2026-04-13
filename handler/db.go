// package handler

// import (
// 	"context"
// 	"log"
// 	"github.com/jackc/pgx/v5"
// 	"github.com/jackc/pgx/v5/pgxpool"
// )

// var db *pgxpool.Pool

// func InitializeDB(connString string) {
// 	config, _ := pgxpool.ParseConfig(connString)
// 	config.MaxConns = 25
// 	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
// 	db, _ = pgxpool.NewWithConfig(context.Background(), config)
// 	log.Println("✅ DB Connected")
// }

// // LogRequest captures technical, financial, and GOVERNANCE telemetry.
// func LogRequest(apiKey string, model string, status int, isHit bool, pTokens int, cTokens int, savings float64, latency int, prompt string, response string, ruleID string, action string) {
// 	if db == nil { return }

// 	// Humne triggered_rule aur governance_action columns add kiye hain
// 	query := `
// 		INSERT INTO request_logs 
// 		(api_key, model, status, is_cache_hit, prompt_tokens, completion_tokens, cost_saved, provider_latency_ms, prompt_text, response_text, triggered_rule, governance_action) 
// 		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
// 	`

// 	go func() {
// 		_, err := db.Exec(context.Background(), query, 
// 			apiKey, model, status, isHit, pTokens, cTokens, savings, latency, prompt, response, ruleID, action,
// 		)
// 		if err != nil {
// 			log.Printf("🚨 Sovereign Logging Failed: %v", err)
// 		}
// 	}()
// }

// func ValidateAPIKey(apiKey string) bool {
// 	var exists bool
// 	db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
// 	return exists
// }

// func CheckUserLimit(apiKey string) (bool, error) {
// 	var used, limit int
// 	err := db.QueryRow(context.Background(), "SELECT requests_used, request_limit FROM users WHERE api_key=$1", apiKey).Scan(&used, &limit)
// 	return used < limit, err
// }

// // func IncrementUsage(apiKey string) {
// // 	go db.Exec(context.Background(), "UPDATE users SET requests_used = requests_used + 1 WHERE api_key=$1", apiKey)
// // }

// func IncrementUsage(apiKey string) {
//     if db == nil { 
//         log.Println("🚨 Sovereign Error: DB Pool is NIL. Usage not tracked.")
//         return 
//     }

//     // 🛡️ THE CRITICAL SYNC: No 'go' keyword. 
//     // We wait for Postgres to confirm the write before the HTTP response ends.
//     // Using context.Background() ensures the update finishes even if the user closes the tab.
//     result, err := db.Exec(context.Background(), 
//         "UPDATE users SET requests_used = requests_used + 1 WHERE api_key=$1", 
//         apiKey,
//     )

//     if err != nil {
//         log.Printf("🚨 SQL Error: Failed to increment usage for %s: %v", apiKey, err)
//         return
//     }

//     // 🔍 INTEGRITY CHECK: Did we actually find this user?
//     rows := result.RowsAffected()
//     if rows == 0 {
//         log.Printf("⚠️ Governance Warning: API Key [%s] not found. No usage recorded.", apiKey)
//     } else {
//         log.Printf("📊 Usage Successfully Incremented for Key: %s", apiKey)
//     }
// }


// func UpgradeUser(apiKey string) {
// 	query := `UPDATE users SET request_limit = 10000, requests_used = 0 WHERE api_key = $1`
// 	_, err := db.Exec(context.Background(), query, apiKey)
// 	if err != nil {
// 		log.Printf("❌ DB Upgrade Fail: %v", err)
// 	} else {
// 		log.Printf("🎉 User %s upgraded to 10k Credits!", apiKey)
// 	}
// }





package handler

import (
    "context"
    "log"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func InitializeDB(connString string) {
    config, _ := pgxpool.ParseConfig(connString)
    config.MaxConns = 25
    config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
    db, _ = pgxpool.NewWithConfig(context.Background(), config)
    log.Println("✅ DB Connected")
}

// IncrementUsage handles the absolute critical billing logic.
func IncrementUsage(apiKey string) {
    if db == nil { 
        log.Println("🚨 Sovereign Error: DB Pool is NIL.")
        return 
    }

    // 🛡️ SYNC EXECUTION: No 'go' routine here to prevent data loss.
    result, err := db.Exec(context.Background(), 
        "UPDATE users SET requests_used = requests_used + 1 WHERE api_key=$1", 
        apiKey,
    )

    if err != nil {
        log.Printf("🚨 SQL Error: Failed to increment usage for %s: %v", apiKey, err)
        return
    }

    if result.RowsAffected() == 0 {
        log.Printf("⚠️ Warning: API Key [%s] not found in Registry.", apiKey)
    } else {
        log.Printf("📊 Usage Successfully Incremented for: %s", apiKey)
    }
}

// LogRequest captures technical, financial, and GOVERNANCE telemetry.
func LogRequest(apiKey string, model string, status int, isHit bool, pTokens int, cTokens int, savings float64, latency int, prompt string, response string, ruleID string, action string) {
    if db == nil { return }

    query := `
        INSERT INTO request_logs 
        (api_key, model, status, is_cache_hit, prompt_tokens, completion_tokens, cost_saved, provider_latency_ms, prompt_text, response_text, triggered_rule, governance_action) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `

    // Telemetry can be async as it doesn't block the quota.
    go func() {
        _, err := db.Exec(context.Background(), query, 
            apiKey, model, status, isHit, pTokens, cTokens, savings, latency, prompt, response, ruleID, action,
        )
        if err != nil {
            log.Printf("🚨 Sovereign Logging Failed: %v", err)
        }
    }()
}

func ValidateAPIKey(apiKey string) bool {
    var exists bool
    db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE api_key=$1)", apiKey).Scan(&exists)
    return exists
}

func CheckUserLimit(apiKey string) (bool, error) {
    var used, limit int
    err := db.QueryRow(context.Background(), "SELECT requests_used, request_limit FROM users WHERE api_key=$1", apiKey).Scan(&used, &limit)
    return used < limit, err
}

func UpgradeUser(apiKey string) {
    query := `UPDATE users SET request_limit = 10000, requests_used = 0 WHERE api_key = $1`
    _, err := db.Exec(context.Background(), query, apiKey)
    if err != nil {
        log.Printf("❌ DB Upgrade Fail: %v", err)
    } else {
        log.Printf("🎉 User %s upgraded to 10k Credits!", apiKey)
    }
}