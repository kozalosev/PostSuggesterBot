package handlers

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/storage"
	"github.com/loctools/go-l10n/loc"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

const (
	approveStatusTrPrefix    = "callbacks.approve.status."
	approveStatusTrRevoked   = approveStatusTrPrefix + "revoked"
	approveStatusTrNoAuthor  = approveStatusTrPrefix + "no.author"
	approveStatusTrPublished = approveStatusTrPrefix + "published"

	approved = "Approved"
)

var (
	requiredApprovals int
	channelID         = parseNotUserID("CHANNEL_ID")
)

func init() {
	if reqApprovals, err := strconv.Atoi(os.Getenv("REQUIRED_APPROVALS")); err == nil {
		requiredApprovals = reqApprovals
	} else {
		log.WithField(logconst.FieldConst, "REQUIRED_APPROVALS").
			Error(err)
		requiredApprovals = 1
	}
}

type ApproveCallbackHandler struct {
	appEnv *base.ApplicationEnv

	approvalService    *repo.ApprovalService
	suggestionsService *repo.SuggestionService
}

func NewApproveCallbackHandler(appEnv *base.ApplicationEnv) *ApproveCallbackHandler {
	return &ApproveCallbackHandler{
		appEnv:             appEnv,
		approvalService:    repo.NewApprovalService(appEnv),
		suggestionsService: repo.NewSuggestionService(appEnv),
	}
}

func (h *ApproveCallbackHandler) GetCallbackPrefix() string {
	return "approve:"
}

func (h *ApproveCallbackHandler) Handle(reqenv *base.RequestEnv, query *tgbotapi.CallbackQuery) {
	role := reqenv.Options.(*dto.UserOptions).Role
	var err error
	if role == dto.UsualUser {
		err = errors.New(approveStatusTrNoAuthor)
	}

	if err == nil {
		var dtoMessage *dto.Message
		data := strings.TrimPrefix(query.Data, h.GetCallbackPrefix())
		if dtoMessage, err = parseMessageParams(data); err == nil {
			var suggestion *dto.Suggestion
			if suggestion, err = h.suggestionsService.Get(dtoMessage); err == nil {
				if suggestion.Revoked {
					err = errors.New(approveStatusTrRevoked)
				} else if err = h.approvalService.Approve(dtoMessage, query.From.ID); err == nil {
					answer := tgbotapi.NewCallback(query.ID, reqenv.Lang.Tr(approved))
					if err = h.appEnv.Bot.Request(answer); err == nil {
						var approvers []string
						approvers, err = h.approvalService.GetApprovers(dtoMessage)
						logEntry := log.WithField(logconst.FieldHandler, "ApproveCallbackHandler").
							WithField(logconst.FieldMethod, "Handle").
							WithField("approvers", approvers).
							WithField("required", requiredApprovals)
						if role == dto.Admin || len(approvers) >= requiredApprovals {
							logEntry.Info("The message is ready to be published!")
							if err = h.publish(suggestion, approvers, query.Message, reqenv.Lang); err == nil {
								err = h.suggestionsService.Publish(dtoMessage)
							}
						} else {
							logEntry.Info("The message is not fully approved yet.")
						}
					}
				}
			}
		}
	}

	var answer tgbotapi.Chattable
	if storage.DuplicateConstraintViolation(err) {
		answer = tgbotapi.NewCallback(query.ID, reqenv.Lang.Tr(approveStatusTrPrefix+duplicate))
	} else if err != nil {
		log.WithField(logconst.FieldHandler, "ApproveCallbackHandler").
			WithField(logconst.FieldMethod, "Handle").
			Error(err)
		answer = tgbotapi.NewCallbackWithAlert(query.ID, reqenv.Lang.Tr(err.Error()))
	} else {
		answer = tgbotapi.NewCallback(query.ID, reqenv.Lang.Tr(success))
	}

	if err := h.appEnv.Bot.Request(answer); err != nil {
		log.WithField(logconst.FieldHandler, "ApproveCallbackHandler").
			WithField(logconst.FieldMethod, "Handle").
			WithField(logconst.FieldCalledObject, "BotAPI").
			WithField(logconst.FieldCalledMethod, "Request").
			Error(err)
	}
}

func (h *ApproveCallbackHandler) publish(suggestion *dto.Suggestion, approvers []string, triggerMsg *tgbotapi.Message, lc *loc.Context) error {
	var forward tgbotapi.Chattable
	if suggestion.Anonymously {
		forward = tgbotapi.NewCopyMessage(channelID, suggestion.UID, suggestion.MessageID)
	} else {
		forward = tgbotapi.NewForward(channelID, suggestion.UID, suggestion.MessageID)
	}
	if err := h.appEnv.Bot.Request(forward); err != nil {
		return err
	}

	newTriggerMsgText := fmt.Sprintf("%s\n\n<i>%s: %s</i>",
		tgbotapi.EscapeText(tgbotapi.ModeHTML, triggerMsg.Text),
		lc.Tr(approveStatusTrPublished),
		tgbotapi.EscapeText(tgbotapi.ModeHTML, strings.Join(approvers, ", ")))
	editTriggerMsg := tgbotapi.NewEditMessageTextAndMarkup(triggerMsg.Chat.ID, triggerMsg.MessageID,
		newTriggerMsgText,
		tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	editTriggerMsg.ParseMode = tgbotapi.ModeHTML
	return h.appEnv.Bot.Request(editTriggerMsg)
}

func parseMessageParams(data string) (*dto.Message, error) {
	arr := strings.Split(data, ":")
	if len(arr) != 2 {
		return nil, errors.New("unexpected format of the callback message: " + data)
	}

	var (
		uid       int64
		messageID int
		err       error
	)
	if uid, err = strconv.ParseInt(arr[0], 10, 64); err == nil {
		if messageID, err = strconv.Atoi(arr[1]); err == nil {
			return dto.NewMessage(uid, messageID), nil
		}
	}
	return nil, err
}
