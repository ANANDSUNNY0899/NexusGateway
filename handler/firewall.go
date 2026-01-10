package handler

import (
	"regexp"
)

// Regex patterns for sensitive data
// 1. Email: looks for text@text.com
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// 2. Phone: looks for 10-digit numbers like 123-456-7890 or (123) 456 7890
var phoneRegex = regexp.MustCompile(`\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)

// RedactPII replaces sensitive info with [REDACTED]
func RedactPII(input string) string {
	// Redact Emails
	input = emailRegex.ReplaceAllString(input, "[EMAIL_REDACTED]")
	
	// Redact Phone Numbers
	input = phoneRegex.ReplaceAllString(input, "[PHONE_REDACTED]")
	
	return input
}