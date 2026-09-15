package brevoimpl

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const (
	defaultHost    = "smtp-relay.brevo.com"
	defaultPort    = "587"
	requestTimeout = 10 * time.Second
)

type Sender struct {
	host string
	port string
	user string
	key  string
	from string
}

func NewSender(host, port, user, key, from string) *Sender {
	return &Sender{
		host: stringOrDefault(host, defaultHost),
		port: stringOrDefault(port, defaultPort),
		user: user,
		key:  key,
		from: from,
	}
}

func (s *Sender) Send(to, subject, body string) error {
	client, err := s.dial()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", s.user, s.key, s.host)); err != nil {
		return err
	}
	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(buildMessage(s.from, to, subject, body)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (s *Sender) dial() (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(s.host, s.port), requestTimeout)
	if err != nil {
		return nil, err
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := client.Hello("localhost"); err != nil {
		_ = client.Close()
		return nil, err
	}

	ok, _ := client.Extension("STARTTLS")
	if !ok {
		_ = client.Close()
		return nil, fmt.Errorf("smtp: server does not support STARTTLS")
	}
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func buildMessage(from, to, subject, body string) []byte {
	var builder strings.Builder
	builder.WriteString("From: " + headerValue(from) + "\r\n")
	builder.WriteString("To: " + headerValue(to) + "\r\n")
	builder.WriteString("Subject: " + headerValue(subject) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return []byte(builder.String())
}

func headerValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	if i := strings.IndexByte(value, '\n'); i >= 0 {
		value = value[:i]
	}
	return value
}

func stringOrDefault(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
