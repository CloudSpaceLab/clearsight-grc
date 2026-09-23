package ropa

import "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"

func newActivityID() (string, error) {
	return id.NewUUIDv7()
}

func newEventID() (string, error) {
	return id.NewUUIDv7()
}
