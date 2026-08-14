package catalog

import "errors"

// ErrNotFound is returned when a game, set, or card doesn't exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when creating something that already exists.
var ErrConflict = errors.New("already exists")

// ErrInvalid is returned for well-formed requests that reference the wrong
// things (e.g. creating a card in a set that doesn't exist).
var ErrInvalid = errors.New("invalid input")
