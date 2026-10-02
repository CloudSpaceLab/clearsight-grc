package controlcatalog

import (
	"context"
	"errors"
)

var (
	ErrInvalid   = errors.New("control catalog input is invalid")
	ErrNotFound  = errors.New("control catalog record not found")
	ErrDuplicate = errors.New("control catalog record already exists")
)

type Repository interface {
	CreateWithImplementationLink(context.Context, Definition, ImplementationLink) (Definition, ImplementationLink, error)
	GetDefinition(context.Context, string, string) (Definition, error)
	LinkImplementation(context.Context, ImplementationLink) (ImplementationLink, error)
	GetImplementationLink(context.Context, string, string, string) (ImplementationLink, error)
	ListImplementationLinks(context.Context, string, string, int) ([]ImplementationLink, error)
}
