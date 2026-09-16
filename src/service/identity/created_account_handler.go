package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajuda-dev/backend/src/client/email"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type createdAccountPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type CreatedAccountHandler struct {
	users  repository.UserRepository
	codes  repository.EmailCodeRepository
	sender email.EmailSender
	cfg    EmailCodeConfig
}

func NewCreatedAccountHandler(
	users repository.UserRepository,
	codes repository.EmailCodeRepository,
	sender email.EmailSender,
	cfg EmailCodeConfig,
) *CreatedAccountHandler {
	if sender == nil {
		sender = email.NewNoopSender()
	}
	return &CreatedAccountHandler{
		users:  users,
		codes:  codes,
		sender: sender,
		cfg:    cfg,
	}
}

func (h *CreatedAccountHandler) Handle(event domain.OutboxEventDomain) error {
	if h == nil || h.users == nil || h.codes == nil {
		return errors.New("created account handler is not configured")
	}
	user, err := h.users.FindById(event.UserId)
	if err != nil {
		return err
	}
	if user.EmailVerified() {
		return nil
	}

	var payload createdAccountPayload
	if len(event.Payload) > 0 {
		if unmarshalErr := json.Unmarshal(event.Payload, &payload); unmarshalErr != nil {
			return unmarshalErr
		}
	}
	to := payload.Email
	if to == "" {
		to = user.Email
	}
	if to == "" {
		return errors.New("created account payload is missing email")
	}

	code, genErr := generateRandomEmailCode()
	if genErr != nil {
		return genErr
	}
	ttl := h.cfg.TTLMinutes
	if ttl <= 0 {
		ttl = defaultEmailCodeTTLMin
	}
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Minute)
	if replaceErr := h.codes.ReplaceActive(
		user.Id,
		domain.EmailCodePurposeConfirm,
		hashEmailConfirmCode(code, h.cfg.Secret),
		expiresAt,
	); replaceErr != nil {
		return replaceErr
	}

	subject := "Confirmação de e-mail"
	body := fmt.Sprintf("Seu código de confirmação de e-mail é: %s\nEste código expira em %d minutos.", code, ttl)
	return h.sender.Send(to, subject, body)
}
