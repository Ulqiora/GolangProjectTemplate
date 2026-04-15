package dql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMessageObject_CreatesPointerModel(t *testing.T) {
	object, err := newMessageObject[*testMessage]()

	require.NoError(t, err)
	require.NotNil(t, object)
}
