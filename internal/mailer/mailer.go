package mailer

import (
	"fmt"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

type Mailer struct {
	client    *mailjet.Client
	fromEmail string
	fromName  string
}

func New(apiKey, secretKey, fromEmail, fromName string) *Mailer {
	client := mailjet.NewMailjetClient(apiKey, secretKey)
	return &Mailer{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (m *Mailer) Send(toEmail, toName, subject, htmlBody string) error {
	messages := mailjet.MessagesV31{
		Info: []mailjet.InfoMessagesV31{
			{
				From: &mailjet.RecipientV31{
					Email: m.fromEmail,
					Name:  m.fromName,
				},
				To: &mailjet.RecipientsV31{
					{
						Email: toEmail,
						Name:  toName,
					},
				},
				Subject:  subject,
				HTMLPart: htmlBody,
			},
		},
	}

	_, err := m.client.SendMailV31(&messages)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}