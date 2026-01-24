package governance

import (
	"log"
	"regexp"
	"strings"
)

type Decision struct {
	Allowed       bool
	ModifiedText  string
	RefusalReason string
	TriggeredRule string
}

// Evaluate performs deterministic checks against the Nexus Constitution
func Evaluate(prompt string) Decision {
	// Default: Allow
	decision := Decision{
		Allowed:      true,
		ModifiedText: prompt,
	}

	// 🛡️ 1. DATA LOSS PREVENTION (DLP)
	// Example: Redacting Patient IDs and Credit Cards
	dlpPatterns := map[string]string{
		"DLP_HOSPITAL": `(?i)(patient_id|ssn|medical_record_number):?\s*[0-9A-Z-]+`,
		"DLP_BANK":     `(?i)(iban|swift|card_number|cvv):?\s*[A-Z0-9]+`,
	}

	for id, pattern := range dlpPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(decision.ModifiedText) {
			log.Printf("🛡️ [GOVERNANCE] Rule %s triggered: Redacting data", id)
			decision.ModifiedText = re.ReplaceAllString(decision.ModifiedText, "[REDACTED_BY_NEXUS]")
			decision.TriggeredRule = id
		}
	}

	// 🛡️ 2. SECURITY GATE (Hard Refusal)
	jailbreakKeywords := []string{"ignore all previous instructions", "system prompt leak"}
	for _, kw := range jailbreakKeywords {
		if strings.Contains(strings.ToLower(prompt), kw) {
			log.Printf("🚫 [GOVERNANCE] Rule SEC_JAILBREAK triggered: Refusing request")
			return Decision{
				Allowed:       false,
				RefusalReason: "Nexus Sovereign Shield: Protocol violation (Unauthorized Instructions).",
				TriggeredRule: "SEC_JAILBREAK",
			}
		}
	}

	// 🛡️ 3. LEGAL COMPLIANCE (Auto-Disclaimer Injection)
	financeKeywords := []string{"bitcoin", "stock", "invest"}
	for _, kw := range financeKeywords {
		if strings.Contains(strings.ToLower(prompt), kw) {
			log.Printf("⚖️ [GOVERNANCE] Rule LEG_FINANCE triggered: Appending Disclaimer")
			decision.TriggeredRule = "LEG_FINANCE"
			// Note: This logic will be applied to the RESPONSE in the stream handler
		}
	}

	return decision
}