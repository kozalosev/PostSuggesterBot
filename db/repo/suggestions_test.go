package repo

import (
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSuggestionService(t *testing.T) {
	clearDatabase(t)
	insertTestSuggestions(t)

	msg1 := dto.NewMessage(TestUID, 1)
	msg2 := dto.NewMessage(TestUID, 2)
	suggestionService := NewSuggestionService(appEnv)

	sug1, err := suggestionService.Get(msg1)
	assert.NoError(t, err)
	assert.Equal(t, TestUID, sug1.UID)
	assert.Equal(t, 1, sug1.MessageID)

	assert.NoError(t, suggestionService.Publish(msg1))
	assert.NoError(t, suggestionService.Revoke(msg2))

	assert.Error(t, suggestionService.Publish(msg2))
	assert.Error(t, suggestionService.Revoke(msg1))
}
