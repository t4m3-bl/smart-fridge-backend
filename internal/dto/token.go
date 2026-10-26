package dto

import (
	"time"
)

type TokenPair struct {
	Refresh Token
	Access  Token
}

type Token struct {
	Body string
	TTL  time.Duration
}

type OAuthTokenPair struct {
	Verifier string
	State    string
	TTL      time.Duration
}
