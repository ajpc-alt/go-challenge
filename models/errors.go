package models

import "errors"

var (
	// ErrNotFound is what repositories report when a lookup matches nothing.
	ErrNotFound = errors.New("not found")

	// ErrDuplicate is what they report when a write collides with a unique
	// constraint.
	ErrDuplicate = errors.New("duplicate")
)
