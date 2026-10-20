package token

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateNonce(len uint) string {
	data := make([]byte, len)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
