package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	_, _, err := Check("tenold.org")
	assert.NoError(t, err)
}
