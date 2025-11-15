package stacktrace

import "sync"

type Pool[T any] struct {
	pool sync.Pool
}

// New Func create wrapped sync pool
func NewPool[T any](fn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return fn()
			},
		},
	}
}

// Get Create object from sync pool
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put Save object from sync pool
func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}
