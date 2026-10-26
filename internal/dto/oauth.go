package dto

import "time"

type OAuthOutput struct {
	State    string
	Verifier string
	TTL      time.Duration
	URL      string
}

type OAuthCallbackInput struct {
	ProviderName string
	Code         string
	Verifier     string
}

type OAuthIdentity struct {
	ProviderUserID string
	Email          string
}
