package email

import (
	"encoding/base64"
	"fmt"
	"html"
	"strings"
	"time"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

type MailjetMailer struct {
	client    *mailjet.Client
	fromEmail string
	fromName  string
}

type TicketIssuedItem struct {
	Code            string
	QRPayload       string
	EventTitle      string
	TicketTypeName  string
	SessionStartsAt time.Time
	SessionEndsAt   time.Time
}

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
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

func (m *MailjetMailer) SendTicketsIssued(toEmail string, toName string, orderNumber string, viewTicketsURL string, tickets []TicketIssuedItem, attachment *EmailAttachment) error {
	toEmail = strings.TrimSpace(toEmail)
	toName = strings.TrimSpace(toName)
	orderNumber = strings.TrimSpace(orderNumber)
	viewTicketsURL = strings.TrimSpace(viewTicketsURL)

	if toEmail == "" {
		return fmt.Errorf("missing recipient email")
	}
	if len(tickets) == 0 {
		return fmt.Errorf("missing ticket items")
	}

	subject := "Your Eventy tickets"
	if orderNumber != "" {
		subject = fmt.Sprintf("Your Eventy tickets (Order %s)", orderNumber)
	}

	textLines := make([]string, 0, len(tickets)+2)
	textLines = append(textLines, "Your tickets are ready.")
	if orderNumber != "" {
		textLines = append(textLines, fmt.Sprintf("Order: %s", orderNumber))
	}
	for _, t := range tickets {
		textLines = append(textLines, fmt.Sprintf("- %s (%s)", t.Code, t.QRPayload))
	}
	if viewTicketsURL != "" {
		textLines = append(textLines, fmt.Sprintf("View your tickets: %s", viewTicketsURL))
	}

	info := mailjet.InfoMessagesV31{
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
		TextPart: strings.Join(textLines, "\n"),
		HTMLPart: buildTicketsIssuedEmailHTML(toEmail, orderNumber, viewTicketsURL, tickets),
	}

	if attachment != nil && len(attachment.Data) > 0 {
		filename := strings.TrimSpace(attachment.Filename)
		if filename == "" {
			filename = "eventy-receipt.pdf"
		}

		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/pdf"
		}

		attachments := mailjet.AttachmentsV31{
			{
				ContentType:   contentType,
				Base64Content: base64.StdEncoding.EncodeToString(attachment.Data),
				Filename:      filename,
			},
		}
		info.Attachments = &attachments
	}

	messages := mailjet.MessagesV31{
		Info: []mailjet.InfoMessagesV31{info},
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

func buildTicketsIssuedEmailHTML(toEmail string, orderNumber string, viewTicketsURL string, tickets []TicketIssuedItem) string {
	safeEmail := html.EscapeString(toEmail)
	safeOrder := html.EscapeString(orderNumber)
	safeViewURL := html.EscapeString(viewTicketsURL)

	rows := make([]string, 0, len(tickets))
	for _, t := range tickets {
		rows = append(rows, fmt.Sprintf(`
        <tr>
          <td style="padding:10px 12px;border-bottom:1px solid #eee;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,'Liberation Mono','Courier New',monospace;">%s</td>
          <td style="padding:10px 12px;border-bottom:1px solid #eee;">%s</td>
          <td style="padding:10px 12px;border-bottom:1px solid #eee;">%s</td>
          <td style="padding:10px 12px;border-bottom:1px solid #eee;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,'Liberation Mono','Courier New',monospace;font-size:12px;color:#555;">%s</td>
        </tr>`,
			html.EscapeString(strings.TrimSpace(t.Code)),
			html.EscapeString(strings.TrimSpace(t.EventTitle)),
			html.EscapeString(strings.TrimSpace(t.TicketTypeName)),
			html.EscapeString(strings.TrimSpace(t.QRPayload)),
		))
	}

	orderLine := ""
	if strings.TrimSpace(orderNumber) != "" {
		orderLine = fmt.Sprintf(`<p style="margin:0 0 12px;color:#666;">Order: <strong style="color:#000;">%s</strong></p>`, safeOrder)
	}

	viewLink := ""
	if strings.TrimSpace(viewTicketsURL) != "" {
		viewLink = fmt.Sprintf(`<p style="margin:16px 0 0;"><a href="%s" style="background:#00bc69;color:#fff;text-decoration:none;padding:10px 14px;border-radius:6px;display:inline-block;">View my tickets</a></p>`, safeViewURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Your tickets</title>
</head>
<body style="margin:0;padding:0;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;color:#333;background-color:#fff;">
  <div style="margin:0 auto;width:100%%;max-width:720px;padding:0 0 10px;border-radius:5px;line-height:1.6;">
    <div style="border-bottom:1px solid #eee;padding-bottom:12px;">
      <div style="font-size:1.4em;color:#000;text-decoration:none;font-weight:600;">Your tickets are ready</div>
    </div>
    <div style="padding-top:20px;">
      %s
      <p style="margin:0 0 12px;color:#666;">Show the ticket code (or a QR built from the payload) at entry.</p>
      <div style="overflow-x:auto;border:1px solid #eee;border-radius:8px;">
        <table style="width:100%%;border-collapse:collapse;font-size:14px;">
          <thead>
            <tr>
              <th align="left" style="padding:10px 12px;border-bottom:1px solid #eee;background:#fafafa;">Code</th>
              <th align="left" style="padding:10px 12px;border-bottom:1px solid #eee;background:#fafafa;">Event</th>
              <th align="left" style="padding:10px 12px;border-bottom:1px solid #eee;background:#fafafa;">Ticket</th>
              <th align="left" style="padding:10px 12px;border-bottom:1px solid #eee;background:#fafafa;">QR payload</th>
            </tr>
          </thead>
          <tbody>
            %s
          </tbody>
        </table>
      </div>
      %s
    </div>
    <hr style="border:none;border-top:0.5px solid #131111;margin:28px 0 14px;" />
    <div style="color:#aaa;font-size:12px;line-height:1.6;font-weight:300;">
      <p style="margin:0 0 6px;">This email was sent to <span style="color:#00bc69;">%s</span>.</p>
      <p style="margin:0;">This inbox is not monitored.</p>
    </div>
  </div>
</body>
</html>`, orderLine, strings.Join(rows, ""), viewLink, safeEmail)
}
