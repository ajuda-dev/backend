package email

// EmailSender é o contrato de envio de e-mail. Service e worker só conhecem
// esta interface; a implementação (Brevo SMTP hoje) fica em xxximpl.
type EmailSender interface {
	Send(to, subject, body string) error
}
