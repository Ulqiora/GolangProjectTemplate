package ports

import "errors"

var (
	ErrNotFound       = errors.New("not a single object was found")
	ErrNofFound       = ErrNotFound
	ErrNoAffectedRows = errors.New("the affected rows were not found")
)
