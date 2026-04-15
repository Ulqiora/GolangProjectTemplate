package ports

import "errors"

var (
	ErrNofFound       = errors.New("not a single object was found")
	ErrNoAffectedRows = errors.New("the affected rows were not found")
)
