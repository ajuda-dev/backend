package email

import (
	"testing"

	"github.com/ajuda-dev/backend/src/client/email/brevoimpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEnvEmptyCredentialsUsesNoopAndDoesNotDial(t *testing.T) {
	t.Setenv(ProviderEnv, "brevo")
	t.Setenv(BrevoHostEnv, "127.0.0.1")
	t.Setenv(BrevoPortEnv, "1")
	t.Setenv(BrevoUserEnv, "")
	t.Setenv(BrevoKeyEnv, "")
	t.Setenv(BrevoFromEnv, "")

	sender := FromEnv()
	require.IsType(t, &noopSender{}, sender)

	err := sender.Send("to@example.com", "subject", "body with a code 123456")
	assert.NoError(t, err)
}

func TestFromEnvDefaultsToBrevoWhenCredentialsAreSet(t *testing.T) {
	t.Setenv(ProviderEnv, "")
	t.Setenv(BrevoUserEnv, "smtp-user")
	t.Setenv(BrevoKeyEnv, "smtp-key")
	t.Setenv(BrevoFromEnv, "noreply@ajuda.dev")

	sender := FromEnv()
	require.IsType(t, &brevoimpl.Sender{}, sender)
}

func TestFromEnvUnknownProviderUsesNoop(t *testing.T) {
	t.Setenv(ProviderEnv, "resend")
	t.Setenv(BrevoUserEnv, "smtp-user")
	t.Setenv(BrevoKeyEnv, "smtp-key")
	t.Setenv(BrevoFromEnv, "noreply@ajuda.dev")

	sender := FromEnv()
	require.IsType(t, &noopSender{}, sender)
}
