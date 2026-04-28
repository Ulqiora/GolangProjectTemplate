package closer

import (
	"errors"
	"testing"
)

func TestGracefulCloser_CloseRunsAllAndJoinsErrors(t *testing.T) {
	t.Parallel()

	errFirst := errors.New("first")
	errSecond := errors.New("second")
	calls := 0

	c := NewGracefulCloser(
		func() error {
			calls++
			return errFirst
		},
		func() error {
			calls++
			return nil
		},
	)
	c.AddCloser(func() error {
		calls++
		return errSecond
	})

	err := c.Close()
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if !errors.Is(err, errFirst) || !errors.Is(err, errSecond) {
		t.Fatalf("expected joined errors, got %v", err)
	}
}
