package evidence

import "context"

type SubjectAccessChecker interface {
	CanReadSubject(context.Context, string, string, string, string) (bool, error)
}

type recipientStore interface {
	CreateRequestWithRecipient(context.Context, Request) (Request, error)
	GetRequestRecipient(context.Context, string, string) (Recipient, error)
	ListRecipientRequests(context.Context, string, string, int) ([]Request, error)
}

func recipientPersistence(repo Repository) (recipientStore, error) {
	store, ok := repo.(recipientStore)
	if !ok {
		return nil, ErrRecipientInvalid
	}
	return store, nil
}

func hydrateRequestRecipient(ctx context.Context, repo Repository, request Request) (Request, error) {
	store, err := recipientPersistence(repo)
	if err != nil {
		return Request{}, err
	}
	recipient, err := store.GetRequestRecipient(ctx, request.TenantID, request.ID)
	if err != nil {
		return Request{}, err
	}
	request.Recipient = recipient
	return request, nil
}

func cloneRecipient(value Recipient) Recipient {
	value.AudienceHash = append([]byte(nil), value.AudienceHash...)
	return value
}
