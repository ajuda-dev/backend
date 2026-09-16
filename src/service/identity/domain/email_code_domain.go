package domain

import "time"

const EmailCodePurposeConfirm = "email_confirm"

type EmailCodeDomain struct {
	Id         string
	UserId     string
	Purpose    string
	CodeHash   string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}
