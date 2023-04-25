package handlers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseMessageParams(t *testing.T) {
	msg, err := parseMessageParams("123456:1")
	assert.NoError(t, err)
	assert.Equal(t, int64(123456), msg.ChatID)
	assert.Equal(t, 1, msg.MessageID)

	msg, err = parseMessageParams("foobar")
	assert.Empty(t, msg)
	assert.Error(t, err)
}
