package handlers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLangFlagToCode(t *testing.T) {
	assert.Equal(t, ruCode, langFlagToCode(ruFlag))
	assert.Equal(t, enCode, langFlagToCode(enFlag))
	assert.Equal(t, "foo", langFlagToCode("foo"))
}
