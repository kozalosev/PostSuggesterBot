package handlers

import (
	_ "embed"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/wizard"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

const (
	helpFieldTrPrefix = "commands.help.fields."

	usernameSubstitution    = "%username%"
	channelNameSubstitution = "%%channelName%%"
)

var (
	//go:embed help.md
	helpMessageEn string

	//go:embed help.ru.md
	helpMessageRu string

	channelName = os.Getenv("CHANNEL_NAME")
)

func init() {
	helpMessageEn = strings.Replace(helpMessageEn, channelNameSubstitution, channelName, 1)
	helpMessageRu = strings.Replace(helpMessageRu, channelNameSubstitution, channelName, 1)
}

type HelpHandler struct {
	base.CommandHandlerTrait

	appEnv       *base.ApplicationEnv
	stateStorage wizard.StateStorage
	langHandler  *LanguageHandler

	userService *repo.UserService
}

func NewHelpHandler(langHandler *LanguageHandler) *HelpHandler {
	h := &HelpHandler{
		appEnv:       langHandler.appEnv,
		stateStorage: langHandler.stateStorage,
		langHandler:  langHandler,
		userService:  langHandler.userService,
	}
	h.HandlerRefForTrait = h
	return h
}

func (*HelpHandler) GetCommands() []string {
	return []string{"help", "start"}
}

func (h *HelpHandler) GetWizardEnv() *wizard.Env {
	return wizard.NewEnv(h.appEnv, h.stateStorage)
}

func (h *HelpHandler) GetWizardDescriptor() *wizard.FormDescriptor {
	desc := wizard.NewWizardDescriptor(h.helpAction)
	lang := desc.AddField(fieldLanguage, helpFieldTrPrefix+fieldLanguage)
	lang.InlineKeyboardAnswers = []string{enFlag, ruFlag}
	return desc
}

func (h *HelpHandler) Handle(reqenv *base.RequestEnv, msg *tgbotapi.Message) {
	username := resolveName(msg.From)
	err := h.userService.Create(msg.From.ID, username)
	if err == nil {
		langForm := wizard.NewWizard(h, 1)
		langForm.AddEmptyField(fieldLanguage, wizard.Text)
		langForm.ProcessNextField(reqenv, msg)
	} else {
		if err != repo.NoRowsWereAffected {
			log.WithField(logconst.FieldHandler, "HelpHandler").
				WithField(logconst.FieldMethod, "Handle").
				WithField(logconst.FieldCalledObject, "UserService").
				WithField(logconst.FieldCalledMethod, "Create").
				Error(err)
		}
		h.sendHelp(msg, reqenv.Lang.GetLanguage())
	}
}

func (h *HelpHandler) helpAction(reqenv *base.RequestEnv, msg *tgbotapi.Message, fields wizard.Fields) {
	h.langHandler.changeLangAction(reqenv, msg, fields)

	lang := fields.FindField(fieldLanguage).Data.(string)
	h.sendHelp(msg, langFlagToCode(lang))
}

func (h *HelpHandler) sendHelp(msg *tgbotapi.Message, langCode string) {
	username := resolveName(msg.From)
	helpText := getHelpMessage(langCode)
	helpText = fillUsername(helpText, username)
	h.appEnv.Bot.ReplyWithMarkdown(msg, helpText)
}

func resolveName(user *tgbotapi.User) string {
	if len(user.UserName) > 0 {
		return user.UserName
	}
	return strings.TrimSpace(user.FirstName + " " + user.LastName)
}

func getHelpMessage(langCode string) string {
	if langCode == ruCode {
		return helpMessageRu
	} else {
		return helpMessageEn
	}
}

func fillUsername(text string, username string) string {
	return strings.Replace(text, usernameSubstitution, username, 1)
}
