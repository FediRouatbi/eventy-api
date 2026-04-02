package email

import (
	"fmt"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

type MailjetMailer struct {
	client    *mailjet.Client
	fromEmail string
	fromName  string
}

func NewMailjetMailer(apiKey string, secretKey string, fromEmail string, fromName string) *MailjetMailer {
	return &MailjetMailer{
		client:    mailjet.NewMailjetClient(apiKey, secretKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (m *MailjetMailer) SendRegistrationOTP(toEmail string, otpCode string) error {
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
					},
				},
				Subject:  "Your Eventy registration code",
				TextPart: fmt.Sprintf("Your Eventy OTP is %s. It expires in a few minutes.", otpCode),
				HTMLPart: fmt.Sprintf("<h3>Your Eventy OTP is %s</h3><p>This code expires in a few minutes.</p>", otpCode),
			},
		},
	}

	_, err := m.client.SendMailV31(&messages)
	return err
}

func (m *MailjetMailer) SendPasswordResetOTP(toEmail string, otpCode string) error {
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
					},
				},
				Subject:  "Your Eventy password reset code",
				TextPart: fmt.Sprintf("Your Eventy password reset OTP is %s. It expires in a few minutes.", otpCode),
				HTMLPart: fmt.Sprintf("<h3>Your Eventy password reset OTP is %s</h3><p>This code expires in a few minutes.</p>", otpCode),
			},
		},
	}

	_, err := m.client.SendMailV31(&messages)
	return err
}
