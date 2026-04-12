package email

import (
	"fmt"
	"html"

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
				TextPart: fmt.Sprintf("Your Eventy verification code is %s. It expires soon. If this wasn't you, ignore this email.", otpCode),
				HTMLPart: buildOTPEmailHTML("Verify your email", "Use this code to finish signing up.", otpCode, toEmail),
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
				TextPart: fmt.Sprintf("Your Eventy password reset code is %s. It expires soon. If this wasn't you, ignore this email.", otpCode),
				HTMLPart: buildOTPEmailHTML("Reset your password", "Use this code to reset your password.", otpCode, toEmail),
			},
		},
	}

	_, err := m.client.SendMailV31(&messages)
	return err
}

func buildOTPEmailHTML(title string, message string, otpCode string, toEmail string) string {
	safeTitle := html.EscapeString(title)
	safeMessage := html.EscapeString(message)
	safeCode := html.EscapeString(otpCode)
	safeEmail := html.EscapeString(toEmail)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
</head>
<body style="margin:0;padding:0;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;color:#333;background-color:#fff;">
  <div style="margin:0 auto;width:100%%;max-width:600px;padding:0 0 10px;border-radius:5px;line-height:1.8;">
    <div style="border-bottom:1px solid #eee;padding-bottom:12px;">
      <div style="font-size:1.4em;color:#000;text-decoration:none;font-weight:600;">%s</div>
    </div>
    <div style="padding-top:20px;">
      <p style="margin:0 0 12px;">%s</p>
      <div style="background:linear-gradient(to right,#00bc69 0,#00bc88 50%%,#00bca8 100%%);margin:18px auto;width:max-content;padding:6px 18px;color:#fff;border-radius:4px;font-size:28px;font-weight:700;letter-spacing:6px;">%s</div>
      <p style="margin:12px 0 0;font-size:14px;color:#666;">Expires in a few minutes. If this wasn't you, ignore this email.</p>
    </div>
    <hr style="border:none;border-top:0.5px solid #131111;margin:28px 0 14px;" />
    <div style="color:#aaa;font-size:12px;line-height:1.6;font-weight:300;">
      <p style="margin:0 0 6px;">This email was sent to <span style="color:#00bc69;">%s</span>.</p>
      <p style="margin:0;">This inbox is not monitored.</p>
    </div>
  </div>
</body>
</html>`, safeTitle, safeTitle, safeMessage, safeCode, safeEmail)
}
