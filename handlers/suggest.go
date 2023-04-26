package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/wizard"
	"github.com/loctools/go-l10n/loc"
	log "github.com/sirupsen/logrus"
)

const (
	suggestFieldTrPrefix  = "handlers.suggest.fields."
	fieldAnonymously      = "anonymously"
	fieldVisibleForAdmins = "visibleForAdmins"
	fieldMessageID        = "messageId"
	fieldConfirmation     = "confirmation"

	yes     = "👍"
	no      = "👎"
	approve = "Approve"
	ban     = "Ban"
	revoke  = "Revoke"

	anon = "visibility.anon"
	pub  = "visibility.public"

	approveMessageTextTr = "messages.approve"
	revokeMessageTextTr  = "messages.revoke"
	refusedMessageTextTr = "messages.refused"
	bannedMessageTextTr  = "messages.banned"
)

var adminChatID = parseNotUserID("ADMIN_CHAT_ID")

type SuggestHandler struct {
	appEnv       *base.ApplicationEnv
	stateStorage wizard.StateStorage

	suggestionService *repo.SuggestionService
}

func NewSuggestHandler(appEnv *base.ApplicationEnv, stateStorage wizard.StateStorage) *SuggestHandler {
	return &SuggestHandler{
		appEnv:            appEnv,
		stateStorage:      stateStorage,
		suggestionService: repo.NewSuggestionService(appEnv),
	}
}

func (h *SuggestHandler) GetWizardEnv() *wizard.Env {
	return wizard.NewEnv(h.appEnv, h.stateStorage)
}

func (h *SuggestHandler) GetWizardDescriptor() *wizard.FormDescriptor {
	desc := wizard.NewWizardDescriptor(h.formAction)

	anonymously := desc.AddField(fieldAnonymously, suggestFieldTrPrefix+fieldAnonymously)
	anonymously.InlineKeyboardAnswers = []string{pub, anon}

	visibleForAdmins := desc.AddField(fieldVisibleForAdmins, suggestFieldTrPrefix+fieldVisibleForAdmins)
	visibleForAdmins.InlineKeyboardAnswers = []string{yes, no}
	visibleForAdmins.SkipIf = wizard.SkipOnFieldValue{
		Name:  fieldAnonymously,
		Value: pub,
	}

	desc.AddField(fieldMessageID, suggestFieldTrPrefix+fieldMessageID)

	confirmation := desc.AddField(fieldConfirmation, suggestFieldTrPrefix+fieldConfirmation)
	confirmation.InlineKeyboardAnswers = []string{yes, no}

	return desc
}

func (*SuggestHandler) CanHandle(_ *base.RequestEnv, msg *tgbotapi.Message) bool {
	return !msg.IsCommand() && msg.Chat.IsPrivate()
}

func (h *SuggestHandler) Handle(reqenv *base.RequestEnv, msg *tgbotapi.Message) {
	reply := base.NewReplier(h.appEnv, reqenv, msg)

	if reqenv.Options.(*dto.UserOptions).Banned {
		reply(bannedMessageTextTr)
	} else {
		w := wizard.NewWizard(h, 4)
		w.AddPrefilledField(fieldMessageID, msg.MessageID)
		w.AddEmptyField(fieldAnonymously, wizard.Text)
		w.AddEmptyField(fieldVisibleForAdmins, wizard.Text)
		w.AddEmptyField(fieldConfirmation, wizard.Text)
		w.ProcessNextField(reqenv, msg)
	}
}

func (h *SuggestHandler) formAction(reqenv *base.RequestEnv, msg *tgbotapi.Message, fields wizard.Fields) {
	confirmation := fields.FindField(fieldConfirmation).Data == yes
	reply := base.NewReplier(h.appEnv, reqenv, msg)
	if !confirmation {
		if err := h.stateStorage.DeleteState(msg.From.ID); err != nil {
			log.WithField(logconst.FieldHandler, "SuggestHandler").
				WithField(logconst.FieldMethod, "formAction").
				WithField(logconst.FieldCalledObject, "StateStorage").
				WithField(logconst.FieldCalledMethod, "DeleteState").
				Error("unable to delete the state: ", err)
		}
		reply(refusedMessageTextTr)
		return
	}

	messageID := int(fields.FindField(fieldMessageID).Data.(float64))
	anonymously := fields.FindField(fieldAnonymously).Data == anon
	visibleForAdmins := fields.FindField(fieldVisibleForAdmins).Data == yes

	messageEntity := dto.NewMessage(msg.From.ID, messageID)
	if err := h.suggestionService.Create(messageEntity, anonymously); err == nil {
		var forward tgbotapi.Chattable
		if anonymously && !visibleForAdmins {
			forward = tgbotapi.NewCopyMessage(adminChatID, msg.Chat.ID, messageID)
		} else {
			forward = tgbotapi.NewForward(adminChatID, msg.Chat.ID, messageID)
		}
		h.replyWithApprovalButtons(forward, msg.From.ID, messageID, reqenv.Lang)
	}

	revokeBtnData := fmt.Sprintf("revoke:%d:%d", msg.Chat.ID, msg.MessageID)
	h.appEnv.Bot.ReplyWithInlineKeyboard(msg, reqenv.Lang.Tr(revokeMessageTextTr), []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(reqenv.Lang.Tr(revoke), revokeBtnData),
	})
}

func (h *SuggestHandler) replyWithApprovalButtons(c tgbotapi.Chattable, authorUID int64, messageID int, lc *loc.Context) {
	if sentMessage, err := h.appEnv.Bot.Send(c); err == nil {
		// CopyMessage returns only MessageID
		if sentMessage.Chat == nil {
			sentMessage.Chat = &tgbotapi.Chat{ID: adminChatID}
		}
		approveCallbackData := fmt.Sprintf("approve:%d:%d", authorUID, messageID)
		banCallbackData := fmt.Sprintf("ban:%d", authorUID)
		h.appEnv.Bot.ReplyWithInlineKeyboard(&sentMessage, lc.Tr(approveMessageTextTr), []tgbotapi.InlineKeyboardButton{
			{Text: lc.Tr(approve), CallbackData: &approveCallbackData},
			{Text: lc.Tr(ban), CallbackData: &banCallbackData},
		})
	} else {
		log.WithField(logconst.FieldHandler, "SuggestHandler").
			WithField(logconst.FieldMethod, "replyWithApprovalButtons").
			WithField(logconst.FieldCalledObject, "BotAPI").
			WithField(logconst.FieldCalledMethod, "Send").
			Error(err)
	}
}
