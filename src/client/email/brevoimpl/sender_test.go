package brevoimpl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMessageIncludesFromToSubject(t *testing.T) {
	msg := string(buildMessage("noreply@ajuda.dev", "user@example.com", "Confirme seu e-mail", "seu codigo e 123456"))

	assert.Contains(t, msg, "From: noreply@ajuda.dev\r\n")
	assert.Contains(t, msg, "To: user@example.com\r\n")
	assert.Contains(t, msg, "Subject: Confirme seu e-mail\r\n")
	assert.Contains(t, msg, "\r\n\r\nseu codigo e 123456")
}

func TestBuildMessageStripsNewlinesFromHeaders(t *testing.T) {
	msg := string(buildMessage(
		"noreply@ajuda.dev",
		"user@example.com\r\nBcc: evil@example.com",
		"assunto\nX-Injected: 1",
		"corpo",
	))

	assert.NotContains(t, msg, "Bcc:")
	assert.NotContains(t, msg, "X-Injected")
	assert.Contains(t, msg, "To: user@example.com\r\n")
	assert.Contains(t, msg, "Subject: assunto\r\n")
}

func TestNewSenderDefaultsHostAndPort(t *testing.T) {
	sender := NewSender("", "", "user", "key", "noreply@ajuda.dev")
	require.NotNil(t, sender)
	assert.Equal(t, defaultHost, sender.host)
	assert.Equal(t, defaultPort, sender.port)
	assert.Equal(t, "smtp-relay.brevo.com", sender.host)
	assert.Equal(t, "587", sender.port)
}
