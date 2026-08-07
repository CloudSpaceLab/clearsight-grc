package workflow

import "context"

type Repository interface {
	List(context.Context, ListFilter) ([]Task, error)
}
