package notifications

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
)

// Dispatcher turns domain events into push notifications. Every method is
// best-effort: failures are logged, never returned, so notification problems
// can never break the business flow that triggered them.
type Dispatcher struct {
	repository *Repository
	sender     *Sender
}

func NewDispatcher(repository *Repository, sender *Sender) *Dispatcher {
	return &Dispatcher{repository: repository, sender: sender}
}

func (d *Dispatcher) enabled() bool {
	return d != nil && d.repository != nil && d.sender != nil && d.sender.client != nil
}

// NotifyUserByEmail pushes to every device registered by a single user.
func (d *Dispatcher) NotifyUserByEmail(ctx context.Context, email, title, body string, data map[string]string) {
	if !d.enabled() {
		return
	}

	tokens, err := d.repository.ListActiveTokensByUserEmail(ctx, email)
	if err != nil {
		log.Printf("[notifications.dispatch] list tokens by email failed: %v", err)
		return
	}

	d.sendToTokens(ctx, tokens, title, body, data)
}

// NotifyEventAudience pushes to every user holding a ticket for the event.
func (d *Dispatcher) NotifyEventAudience(ctx context.Context, eventID uuid.UUID, title, body string, data map[string]string) {
	if !d.enabled() {
		return
	}

	tokens, err := d.repository.ListActiveTokensByEventID(ctx, eventID)
	if err != nil {
		log.Printf("[notifications.dispatch] list tokens by event failed: %v", err)
		return
	}

	d.sendToTokens(ctx, tokens, title, body, data)
}

// NotifyAllUsers broadcasts to every registered device.
func (d *Dispatcher) NotifyAllUsers(ctx context.Context, title, body string, data map[string]string) {
	if !d.enabled() {
		return
	}

	tokens, err := d.repository.ListAllActiveTokens(ctx)
	if err != nil {
		log.Printf("[notifications.dispatch] list all tokens failed: %v", err)
		return
	}

	d.sendToTokens(ctx, tokens, title, body, data)
}

func (d *Dispatcher) sendToTokens(ctx context.Context, tokens []string, title, body string, data map[string]string) {
	for _, token := range tokens {
		if _, err := d.sender.SendToToken(ctx, token, title, body, data); err != nil {
			// A token Firebase no longer recognizes will never work again, so
			// revoke it instead of retrying it on every future send.
			if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
				if revokeErr := d.repository.RevokeToken(ctx, token); revokeErr != nil {
					log.Printf("[notifications.dispatch] revoke stale token failed: %v", revokeErr)
				}
				continue
			}

			log.Printf("[notifications.dispatch] send failed: %v", err)
		}
	}
}
