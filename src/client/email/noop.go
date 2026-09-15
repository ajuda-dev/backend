package email

type noopSender struct{}

func NewNoopSender() EmailSender {
	return &noopSender{}
}

func (n *noopSender) Send(_, _, _ string) error {
	return nil
}
