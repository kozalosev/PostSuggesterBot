package handlers

import (
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	log "github.com/sirupsen/logrus"
	"strings"
)

const revokeStatusPublished = "callbacks.revoke.status.published"

type RevokeCallbackHandler struct {
	appEnv *base.ApplicationEnv

	suggestionService *repo.SuggestionService
}

func NewRevokeCallbackHandler(appEnv *base.ApplicationEnv) *RevokeCallbackHandler {
	return &RevokeCallbackHandler{
		appEnv:            appEnv,
		suggestionService: repo.NewSuggestionService(appEnv),
	}
}

func (*RevokeCallbackHandler) GetCallbackPrefix() string {
	return "revoke:"
}

func (h *RevokeCallbackHandler) Handle(reqenv *base.RequestEnv, query *tgbotapi.CallbackQuery) {
	var err error
	data := strings.TrimPrefix(query.Data, h.GetCallbackPrefix())
	if msg, e := parseMessageParams(data); e == nil {
		err = h.suggestionService.Revoke(msg)
	} else {
		err = e
	}

	var answer tgbotapi.Chattable
	if attemptToRevokePublished(err) {
		answer = tgbotapi.NewCallbackWithAlert(query.ID, reqenv.Lang.Tr(revokeStatusPublished))
	} else if err != nil {
		log.WithField(logconst.FieldHandler, "RevokeCallbackHandler").
			WithField(logconst.FieldMethod, "Handler").
			WithField(logconst.FieldCalledObject, "SuggestionService").
			WithField(logconst.FieldCalledMethod, "Revoke").
			Error(err)
		answer = tgbotapi.NewCallbackWithAlert(query.ID, reqenv.Lang.Tr(failure))
	} else {
		answer = tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID,
			query.Message.Text+"\n\n"+reqenv.Lang.Tr("Revoked"),
			tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	}
	if err := h.appEnv.Bot.Request(answer); err != nil {
		log.WithField(logconst.FieldHandler, "RevokeCallbackHandler").
			WithField(logconst.FieldMethod, "Handler").
			WithField(logconst.FieldCalledObject, "BotAPI").
			WithField(logconst.FieldCalledMethod, "Request").
			Error(err)
	}
}

func attemptToRevokePublished(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Message == "The suggestion cannot be revoked and published simultaneously!"
}
