package email

import (
	gomail "gopkg.in/mail.v2"
)

type SMTPEmailManager struct {
	cfg    *Config
	dialer *gomail.Dialer
}

type Manager interface {
	SendRegistrationNotification(message *Message) error
}

func (e SMTPEmailManager) SendRegistrationNotification(message *Message) error {
	messageSmtp := gomail.NewMessage()
	messageSmtp.SetHeader("From", e.cfg.Email)
	messageSmtp.SetHeader("To", message.To)
	messageSmtp.SetHeader("Subject", message.Subject)

	messageSmtp.SetBody(ContentTypeTextPlain, message.Body.(string))
	return e.dialer.DialAndSend(messageSmtp)
}

func NewEmailManager(config *Config) Manager {
	return &SMTPEmailManager{
		cfg:    config,
		dialer: gomail.NewDialer(config.Host, config.Port, config.Email, config.Password),
	}
}
