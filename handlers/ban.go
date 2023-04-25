package handlers

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

const adminOnlyMessageTr = "messages.admin.only"

type BanCallbackHandler struct {
	appEnv *base.ApplicationEnv

	userService *repo.UserService
}

func NewBanCallbackHandler(appEnv *base.ApplicationEnv) *BanCallbackHandler {
	return &BanCallbackHandler{
		appEnv:      appEnv,
		userService: repo.NewUserService(appEnv),
	}
}

func (h *BanCallbackHandler) GetCallbackPrefix() string {
	return "ban:"
}

func (h *BanCallbackHandler) Handle(reqenv *base.RequestEnv, query *tgbotapi.CallbackQuery) {
	if !h.ensureUserIsAdmin(query, reqenv) {
		return
	}

	var (
		markup tgbotapi.InlineKeyboardMarkup
		err    error
	)
	data := strings.TrimPrefix(query.Data, h.GetCallbackPrefix())
	if uid, unban, e := parseBanCallbackData(data); e == nil {
		markup = *query.Message.ReplyMarkup
		banBtn := &markup.InlineKeyboard[0][1]

		if unban {
			err = h.userService.Unban(uid)

			banBtn.Text = reqenv.Lang.Tr("Ban")
			banBtnData := fmt.Sprintf("ban:%d", uid)
			banBtn.CallbackData = &banBtnData
		} else {
			err = h.userService.Ban(uid)

			banBtn.Text = reqenv.Lang.Tr("Unban")
			banBtnData := fmt.Sprintf("ban:un:%d", uid)
			banBtn.CallbackData = &banBtnData
		}
	} else {
		err = e
	}

	var answer tgbotapi.Chattable
	if err != nil {
		answer = tgbotapi.NewCallbackWithAlert(query.ID, reqenv.Lang.Tr(failure))
	} else {
		answer = tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, markup)
	}

	if err := h.appEnv.Bot.Request(answer); err != nil {
		log.WithField(logconst.FieldHandler, "BanCallbackHandler").
			WithField(logconst.FieldMethod, "Handle").
			WithField(logconst.FieldCalledObject, "BotAPI").
			WithField(logconst.FieldCalledMethod, "Request").
			Error(err)
	}
}

func (h *BanCallbackHandler) ensureUserIsAdmin(query *tgbotapi.CallbackQuery, reqenv *base.RequestEnv) bool {
	if reqenv.Options.(*dto.UserOptions).Role != dto.Admin {
		rejection := tgbotapi.NewCallbackWithAlert(query.ID, reqenv.Lang.Tr(adminOnlyMessageTr))
		if err := h.appEnv.Bot.Request(rejection); err != nil {
			log.WithField(logconst.FieldHandler, "BanCallbackHandler").
				WithField(logconst.FieldMethod, "Handle").
				WithField(logconst.FieldCalledObject, "BotAPI").
				WithField(logconst.FieldCalledMethod, "Request").
				Error(err)
		}
		return false
	}
	return true
}

func parseBanCallbackData(data string) (uid int64, unban bool, err error) {
	arr := strings.Split(data, callbackDataSep)
	switch len(arr) {
	case 1:
		uid, err = strconv.ParseInt(arr[0], 10, 64)
	case 2:
		unban = arr[0] == "un"
		uid, err = strconv.ParseInt(arr[1], 10, 64)
	default:
		err = errors.New("unexpected format of the callback data")
	}
	return
}
