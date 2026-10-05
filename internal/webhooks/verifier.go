package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifyHMACSHA256 verifies that signature matches the HMAC SHA256 of payload with secret.
func VerifyHMACSHA256(payload []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	// Remove possible prefix like "sha256="
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedHex := hex.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signature), []byte(expectedHex))
}
