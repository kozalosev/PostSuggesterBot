package handlers

import (
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/wizard"
	log "log/slog"
)

// NotPrivateChatFallbackHandler is a guard against accidental execution of the wizard in the admin chat.
type NotPrivateChatFallbackHandler struct {
	stateStorage wizard.StateStorage
}

func NewNotPrivateChatFallbackHandler(stateStorage wizard.StateStorage) *NotPrivateChatFallbackHandler {
	return &NotPrivateChatFallbackHandler{stateStorage: stateStorage}
}

func (f *NotPrivateChatFallbackHandler) CanHandle(_ *base.RequestEnv, msg *tgbotapi.Message) bool {
	return !msg.Chat.IsPrivate()
}

func (f *NotPrivateChatFallbackHandler) Handle(_ *base.RequestEnv, msg *tgbotapi.Message) {
	if err := f.stateStorage.DeleteState(msg.From.ID); err != nil {
		log.Error("Failed to delete the state",
			logconst.FieldHandler, "NotPrivateChatFallbackHandler",
			logconst.FieldMethod, "Handle",
			logconst.FieldCalledObject, "StateStorage",
			logconst.FieldCalledMethod, "DeleteState",
			"UID", msg.From.ID,
			logconst.FieldError, err)
	}
}
