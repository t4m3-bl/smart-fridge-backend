package sha256

import "crypto/sha256"

func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return string(sum[:])
}
