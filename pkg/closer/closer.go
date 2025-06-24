package closer

import "errors"

type Closer interface {
	Close() error
}

type CloseFunc func() error

type GracefulCloser struct {
	CloseObjects []CloseFunc
}

func NewGracefulCloser(closers ...CloseFunc) *GracefulCloser {
	object := GracefulCloser{}
	object.CloseObjects = append(object.CloseObjects, closers...)
	return &object
}

func (object *GracefulCloser) Close() error {
	var errs []error
	for _, closer := range object.CloseObjects {
		err := closer()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (object *GracefulCloser) AddCloser(closers ...CloseFunc) {
	object.CloseObjects = append(object.CloseObjects, closers...)
}
