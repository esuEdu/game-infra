package domain

import "errors"

var (
	ErrNoActiveGame    = errors.New("no active game")
	ErrUnknownGameType = errors.New("unknown game type")
	ErrAnotherInFlight = errors.New("another operation is in progress")
	ErrBadState        = errors.New("invalid state")
)
