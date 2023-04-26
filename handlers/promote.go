package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/wizard"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

const (
	promoteFieldsTrPrefix = "commands.promote.fields."
	fieldUID              = "uid"
	fieldRole             = "role"
	fieldAutoAdmins       = "autoAdmins"

	promoteStatusTrSuccess = "commands.promote.status.success"
	promoteStatusTrNoOne   = "commands.promote.status.nobody"
)

type PromoteHandler struct {
	appEnv       *base.ApplicationEnv
	stateStorage wizard.StateStorage

	userService *repo.UserService
}

func NewPromoteHandler(appEnv *base.ApplicationEnv, stateStorage wizard.StateStorage) *PromoteHandler {
	return &PromoteHandler{
		appEnv:       appEnv,
		stateStorage: stateStorage,
		userService:  repo.NewUserService(appEnv),
	}
}

func (h *PromoteHandler) GetWizardEnv() *wizard.Env {
	return wizard.NewEnv(h.appEnv, h.stateStorage)
}

func (h *PromoteHandler) GetWizardDescriptor() *wizard.FormDescriptor {
	desc := wizard.NewWizardDescriptor(h.formAction)

	uid := desc.AddField(fieldUID, promoteFieldsTrPrefix+fieldUID)
	uid.SkipIf = wizard.SkipIfFiledNotEmpty{Name: fieldAutoAdmins}

	role := desc.AddField(fieldRole, promoteFieldsTrPrefix+fieldRole)
	role.InlineKeyboardAnswers = []string{string(dto.UsualUser), string(dto.Author), string(dto.Admin)}

	autoAdmins := desc.AddField(fieldAutoAdmins, promoteFieldsTrPrefix+fieldAutoAdmins)
	autoAdmins.InlineKeyboardAnswers = []string{yes, no}
	autoAdmins.SkipIf = wizard.SkipIfFiledNotEmpty{Name: fieldUID}

	return desc
}

func (*PromoteHandler) CanHandle(reqenv *base.RequestEnv, msg *tgbotapi.Message) bool {
	return msg.Command() == "promote" && reqenv.Options.(*dto.UserOptions).Role == dto.Admin
}

func (h *PromoteHandler) Handle(reqenv *base.RequestEnv, msg *tgbotapi.Message) {
	form := wizard.NewWizard(h, 3)
	form.AddEmptyField(fieldAutoAdmins, wizard.Text)

	if msg.ReplyToMessage != nil {
		form.AddPrefilledField(fieldUID, msg.ReplyToMessage.From.ID)
	} else {
		form.AddEmptyField(fieldUID, wizard.Text)
	}

	if arg := base.GetCommandArgument(msg); len(arg) > 0 {
		form.AddPrefilledField(fieldRole, arg)
	} else {
		form.AddEmptyField(fieldRole, wizard.Text)
	}

	form.ProcessNextField(reqenv, msg)
}

func (h *PromoteHandler) formAction(reqenv *base.RequestEnv, msg *tgbotapi.Message, fields wizard.Fields) {
	var uid float64
	if uidData := fields.FindField(fieldUID).Data; uidData != nil {
		uid = uidData.(float64)
	}
	autoAdmins := fields.FindField(fieldAutoAdmins).Data == yes
	role := dto.UserRole(fields.FindField(fieldRole).Data.(string))

	candidates := h.resolveCandidates(uid, resolveName(msg.From), autoAdmins)
	log.WithField(logconst.FieldHandler, "PromoteHandler").
		WithField(logconst.FieldMethod, "formAction").
		Infof("I'm going to promote %v to the %s role", candidates, role)

	errs := funk.Map(candidates, func(c candidate) error {
		if err := h.userService.Promote(c.uid, role); err == repo.NoRowsWereAffected {
			if err := h.userService.Create(c.uid, c.name); err != nil && err != repo.NoRowsWereAffected {
				return err
			}
			return h.userService.Promote(c.uid, role)
		} else {
			return err
		}
	}).([]error)
	errs = funk.Filter(errs, func(e error) bool {
		return e != nil
	}).([]error)
	for _, e := range errs {
		log.WithField(logconst.FieldHandler, "PromoteHandler").
			WithField(logconst.FieldMethod, "formAction").
			WithField(logconst.FieldCalledObject, "UserService").
			WithField(logconst.FieldCalledMethod, "Promote").
			Error("unable to promote the user", e)
	}

	if err := h.stateStorage.DeleteState(msg.From.ID); err != nil {
		log.WithField(logconst.FieldHandler, "PromoteHandler").
			WithField(logconst.FieldMethod, "formAction").
			WithField(logconst.FieldCalledObject, "StateStorage").
			WithField(logconst.FieldCalledMethod, "DeleteState").
			Error("unable to delete the state: ", err)
	}

	reply := base.NewReplier(h.appEnv, reqenv, msg)
	if len(errs) > 0 {
		reply(failure)
	} else if len(candidates) > 0 {
		names := funk.Reduce(candidates[1:], func(acc string, c candidate) string {
			return fmt.Sprintf("%s, %s", acc, c.name)
		}, candidates[0].name).(string)
		h.appEnv.Bot.Reply(msg, reqenv.Lang.Tr(promoteStatusTrSuccess)+names)
	} else {
		reply(promoteStatusTrNoOne)
	}
}

func (h *PromoteHandler) resolveCandidates(uid float64, name string, autoAdmins bool) []candidate {
	if uid == 0 && autoAdmins {
		if ids, err := h.fetchAdmins(); err == nil {
			return ids
		} else {
			log.WithField(logconst.FieldHandler, "PromoteHandler").
				WithField(logconst.FieldMethod, "formAction").
				WithField(logconst.FieldCalledMethod, "fetchAdmins").
				Error("unable to fetch UIDs of the chat administrators", err)
		}
	} else if uid > 0 {
		return []candidate{{uid: int64(uid), name: name}}
	}
	return nil
}

func (h *PromoteHandler) fetchAdmins() ([]candidate, error) {
	reqConfig := tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: adminChatID}}
	admins, err := h.appEnv.Bot.GetStandardAPI().GetChatAdministrators(reqConfig)
	return funk.Map(admins, func(u tgbotapi.ChatMember) candidate {
		return candidate{
			uid:  u.User.ID,
			name: resolveName(u.User),
		}
	}).([]candidate), err
}

// candidate for promotion to another role
type candidate struct {
	uid  int64
	name string
}
