package thirdparty

import (
	"context"
	"errors"
)

var (
	ErrInvalid         = errors.New("invalid third-party relationship")
	ErrNotFound        = errors.New("third-party relationship not found")
	ErrVersionConflict = errors.New("third-party relationship version conflict")
)

type Repository interface {
	CreateRelationship(context.Context, CreateRecord) (Aggregate, error)
	UpdateRelationship(context.Context, UpdateRecord) (Aggregate, error)
	GetRelationship(context.Context, Scope, string) (Aggregate, error)
	ListRelationships(context.Context, ListFilter) (RelationshipPage, error)
}
