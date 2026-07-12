// mint-token-debug — quick standalone JWT minter with the M1 test secret.
// Used to debug 401 responses by hand.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	secret := "dev-secret-m1-7f8a-2026"
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().UTC()
	payload := map[string]any{
		"sub": "alice-0",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	fmt.Println(signing + "." + sig)
}
