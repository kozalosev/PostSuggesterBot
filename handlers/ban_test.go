package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/loctools/go-l10n/loc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseBanCallbackData(t *testing.T) {
	data4ban := "123456"
	data4unban := "un:123456"
	invalidData := "foobar"

	uid, unban, err := parseBanCallbackData(data4ban)
	assert.NoError(t, err)
	assert.Equal(t, int64(123456), uid)
	assert.False(t, unban)

	uid, unban, err = parseBanCallbackData(data4unban)
	assert.NoError(t, err)
	assert.Equal(t, int64(123456), uid)
	assert.True(t, unban)

	_, _, err = parseBanCallbackData(invalidData)
	assert.Error(t, err)
}

func TestEnsureUserIsAdmin(t *testing.T) {
	bot := &base.FakeBotAPI{}
	opts := &dto.UserOptions{Role: dto.Author}
	appEnv := &base.ApplicationEnv{Bot: bot}
	reqEnv := &base.RequestEnv{
		Lang:    loc.NewPool("en").GetContext("en"),
		Options: opts,
	}
	query := &tgbotapi.CallbackQuery{ID: "1"}

	h := NewBanCallbackHandler(appEnv)
	assert.False(t, h.ensureUserIsAdmin(query, reqEnv))
	assert.Len(t, bot.GetOutput(), 1)

	opts.Role = dto.Admin
	bot.ClearOutput()
	assert.True(t, h.ensureUserIsAdmin(query, reqEnv))
	assert.Empty(t, bot.GetOutput())
}
