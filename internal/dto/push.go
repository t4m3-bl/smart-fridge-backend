package dto

type PushSubscriptionInput struct {
	LogoutTokenHash *string
	Endpoint        string
	P256DH          string
	Auth            string
}
