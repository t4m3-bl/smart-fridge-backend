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
