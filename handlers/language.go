package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/settings"
	"github.com/kozalosev/goSadTgBot/wizard"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const (
	langFieldsTrPrefix = "commands.language.fields."

	fieldLanguage = "language"

	enCode = "en"
	enFlag = "🇺🇸"
	ruCode = "ru"
	ruFlag = "🇷🇺"
)

var supportedLangCodes = []string{enFlag, enCode, ruFlag, ruCode}

type LanguageHandler struct {
	base.CommandHandlerTrait

	appEnv       *base.ApplicationEnv
	stateStorage wizard.StateStorage

	userService *repo.UserService
}

func NewLanguageHandler(appEnv *base.ApplicationEnv, stateStorage wizard.StateStorage) *LanguageHandler {
	h := &LanguageHandler{
		appEnv:       appEnv,
		stateStorage: stateStorage,
		userService:  repo.NewUserService(appEnv),
	}
	h.HandlerRefForTrait = h
	return h
}

func (h *LanguageHandler) GetWizardEnv() *wizard.Env {
	return wizard.NewEnv(h.appEnv, h.stateStorage)
}

func (h *LanguageHandler) GetWizardDescriptor() *wizard.FormDescriptor {
	desc := wizard.NewWizardDescriptor(h.changeLangAction)
	lang := desc.AddField(fieldLanguage, langFieldsTrPrefix+fieldLanguage)
	lang.InlineKeyboardAnswers = []string{enFlag, ruFlag}
	return desc
}

func (*LanguageHandler) GetCommands() []string {
	return []string{"language", "lang"}
}

func (h *LanguageHandler) Handle(reqenv *base.RequestEnv, msg *tgbotapi.Message) {
	arg := base.GetCommandArgument(msg)

	langForm := wizard.NewWizard(h, 1)
	if len(arg) > 0 && slices.Contains(supportedLangCodes, arg) {
		langForm.AddPrefilledField(fieldLanguage, arg)
	} else {
		langForm.AddEmptyField(fieldLanguage, wizard.Text)
	}
	langForm.ProcessNextField(reqenv, msg)
}

func (h *LanguageHandler) changeLangAction(reqenv *base.RequestEnv, msg *tgbotapi.Message, fields wizard.Fields) {
	langFlag := fields.FindField(fieldLanguage).Data.(string)
	langCode := langFlagToCode(langFlag)
	reply := base.NewReplier(h.appEnv, reqenv, msg)

	err := h.userService.ChangeLanguage(msg.From.ID, settings.LangCode(langCode))
	if err != nil {
		log.WithField(logconst.FieldHandler, "LanguageHandler").
			WithField(logconst.FieldMethod, "changeLangAction").
			WithField(logconst.FieldCalledObject, "UserService").
			WithField(logconst.FieldCalledMethod, "ChangeLanguage").
			Error(err)
		reply(failure)
	} else {
		reply(success)
	}
}

func langFlagToCode(flag string) string {
	switch flag {
	case enFlag:
		return enCode
	case ruFlag:
		return ruCode
	default:
		return flag
	}
}
