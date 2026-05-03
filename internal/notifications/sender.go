package notifications

import (
	"context"
	"errors"

	"firebase.google.com/go/v4/messaging"
)

type Sender struct {
	client *messaging.Client
}

func NewSender(client *messaging.Client) *Sender {
	return &Sender{client: client}
}

func (s *Sender) SendToToken(ctx context.Context, token string, title string, body string, data map[string]string) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("firebase messaging is not configured")
	}

	return s.client.Send(ctx, &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	})
}
