package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/wizard"
	log "github.com/sirupsen/logrus"
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
		log.WithField(logconst.FieldHandler, "NotPrivateChatFallbackHandler").
			WithField(logconst.FieldMethod, "Handle").
			WithField(logconst.FieldCalledObject, "StateStorage").
			WithField(logconst.FieldCalledMethod, "DeleteState").
			WithField("UID", msg.From.ID).
			Error("unable to delete the state: ", err)
	}
}
