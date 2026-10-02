package risk

import "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"

func newID() (string, error) { return id.NewUUIDv7() }
