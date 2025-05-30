package kafka

import (
	"context"
	"sync"
)

type Runnable interface {
	Run(ctx context.Context, group *sync.WaitGroup) error
}
