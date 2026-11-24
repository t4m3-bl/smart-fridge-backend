package main

import (
	"encoding/json"
	"log"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Generate once per environment and store both values as deployment secrets.
func main() {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatal(err)
	}
	// #nosec G117 -- This explicit key-generation command intentionally prints the newly generated key pair.
	if err = json.NewEncoder(os.Stdout).Encode(struct {
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	}{public, private}); err != nil {
		log.Fatal(err)
	}
}
