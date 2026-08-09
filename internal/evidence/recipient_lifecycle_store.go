package evidence

import (
	"context"
	"time"
)

type recipientLifecycleStore interface {
	DeclareWrongRecipient(context.Context, DeclareWrongRecipientInput, time.Time) error
	ReassignRecipient(context.Context, ReassignRecipientInput, Recipient, time.Time) error
}

func recipientLifecyclePersistence(repo Repository) (recipientLifecycleStore, error) {
	store, ok := repo.(recipientLifecycleStore)
	if !ok {
		return nil, ErrRecipientInvalid
	}
	return store, nil
}
