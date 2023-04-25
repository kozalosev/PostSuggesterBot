package repo

import (
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApprovalService(t *testing.T) {
	clearDatabase(t)
	insertTestSuggestions(t)

	msg := dto.NewMessage(TestUID, 1)
	approvalService := NewApprovalService(appEnv)

	approvers, err := approvalService.GetApprovers(msg)
	assert.NoError(t, err)
	assert.Len(t, approvers, 0)

	assert.NoError(t, approvalService.Approve(msg, TestUID))

	approvers, err = approvalService.GetApprovers(msg)
	assert.NoError(t, err)
	assert.Len(t, approvers, 1)
	assert.Contains(t, approvers, TestUser)
}

func insertTestSuggestions(t *testing.T) {
	userService := NewUserService(appEnv)
	assert.NoError(t, userService.Create(TestUID, TestUser))
	assert.NoError(t, userService.Promote(TestUID, dto.Author))

	suggestionService := NewSuggestionService(appEnv)
	assert.NoError(t, suggestionService.Create(dto.NewMessage(TestUID, 1), false))
	assert.NoError(t, suggestionService.Create(dto.NewMessage(TestUID, 2), true))
}
