package models

import "errors"

// ErrNotFound is what repositories report when a lookup matches nothing. It
// keeps GORM's own sentinel from leaking into the handlers.
var ErrNotFound = errors.New("not found")
