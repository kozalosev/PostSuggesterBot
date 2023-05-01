package handlers

import (
	"fmt"
	"github.com/butuzov/harmony"
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
	fieldName             = "name"
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

	name := desc.AddField(fieldName, promoteFieldsTrPrefix+fieldName)
	name.SkipIf = wizard.SkipIfFiledNotEmpty{Name: fieldAutoAdmins}

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
		user := msg.ReplyToMessage.From
		form.AddPrefilledField(fieldUID, user.ID)
		form.AddPrefilledField(fieldName, resolveName(user))
	} else {
		form.AddEmptyField(fieldUID, wizard.Text)
		form.AddEmptyField(fieldName, wizard.Text)
	}

	if arg := base.GetCommandArgument(msg); len(arg) > 0 {
		form.AddPrefilledField(fieldRole, arg)
	} else {
		form.AddEmptyField(fieldRole, wizard.Text)
	}

	form.ProcessNextField(reqenv, msg)
}

func (h *PromoteHandler) formAction(reqenv *base.RequestEnv, msg *tgbotapi.Message, fields wizard.Fields) {
	var (
		uid      float64
		username string
	)
	if uidData := fields.FindField(fieldUID).Data; uidData != nil {
		uid = uidData.(float64)
		username = fields.FindField(fieldName).Data.(string)
	}
	autoAdmins := fields.FindField(fieldAutoAdmins).Data == yes
	role := dto.UserRole(fields.FindField(fieldRole).Data.(string))

	candidates := h.resolveCandidates(uid, username, autoAdmins)
	log.WithField(logconst.FieldHandler, "PromoteHandler").
		WithField(logconst.FieldMethod, "formAction").
		Infof("I'm going to promote %s to the %s role", candidates, role)

	candidates = funk.Filter(candidates, func(c *candidate) bool {
		return c.currRole != dto.Admin || !autoAdmins
	}).([]*candidate)
	errs := funk.Map(candidates, func(c *candidate) error {
		if !c.exist {
			if err := h.userService.Create(c.uid, c.name); err != nil {
				return err
			}
		}
		if c.currRole != role {
			return h.userService.Promote(c.uid, role)
		} else {
			log.WithField(logconst.FieldHandler, "PromoteHandler").
				WithField(logconst.FieldMethod, "formAction").
				Infof("%s has %s role already", c.name, role)
			return nil
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
		names := funk.Reduce(candidates[1:], func(acc string, c *candidate) string {
			return fmt.Sprintf("%s, %s", acc, c.name)
		}, candidates[0].name).(string)
		h.appEnv.Bot.Reply(msg, reqenv.Lang.Tr(promoteStatusTrSuccess)+names)
	} else {
		reply(promoteStatusTrNoOne)
	}
}

func (h *PromoteHandler) resolveCandidates(uid float64, username string, autoAdmins bool) []*candidate {
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
		return []*candidate{<-h.fetchUserInfo(int64(uid), username)}
	}
	return nil
}

func (h *PromoteHandler) fetchAdmins() ([]*candidate, error) {
	channels := funk.Map([]int64{adminChatID, channelID}, func(chatID int64) <-chan *candidate {
		return h.fetchAdminsForChat(chatID)
	}).([]<-chan *candidate)

	done := make(chan struct{})
	ch, err := harmony.FanIn(done, channels[0], channels[1])
	if err != nil {
		return nil, err
	}

	var candidates []*candidate
	for c := range ch {
		candidates = append(candidates, c)
	}
	return candidates, nil
}

func (h *PromoteHandler) fetchAdminsForChat(chatID int64) <-chan *candidate {
	ch := make(chan *candidate)
	go func() {
		defer close(ch)

		reqConfig := tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}}
		if members, err := h.appEnv.Bot.GetStandardAPI().GetChatAdministrators(reqConfig); err == nil {
			members = funk.Filter(members, func(m tgbotapi.ChatMember) bool {
				return !m.User.IsBot
			}).([]tgbotapi.ChatMember)

			switch len(members) {
			case 0:
				break
			case 1:
				ch <- <-h.fetchUserInfo(members[0].User.ID, resolveName(members[0].User))
			default:
				done := make(chan struct{})
				rest := funk.Map(members[2:], func(member tgbotapi.ChatMember) <-chan *candidate {
					return h.fetchUserInfo(member.User.ID, resolveName(member.User))
				}).([]<-chan *candidate)
				c, err := harmony.FanIn(done,
					h.fetchUserInfo(members[0].User.ID, resolveName(members[0].User)),
					h.fetchUserInfo(members[1].User.ID, resolveName(members[1].User)),
					rest...)
				if err == nil {
					for d := range c {
						ch <- d
					}
				} else {
					log.WithField(logconst.FieldHandler, "PromoteHandler").
						WithField(logconst.FieldMethod, "fetchAdminsForChat").
						WithField(logconst.FieldCalledFunc, "FanIn").
						Error(err)
				}
			}
		} else {
			log.WithField(logconst.FieldHandler, "PromoteHandler").
				WithField(logconst.FieldMethod, "fetchAdminsForChat").
				WithField(logconst.FieldCalledObject, "BotAPI").
				WithField(logconst.FieldCalledMethod, "GetChatAdministrators").
				Error("unable to get the list of administrators", err)
		}
	}()
	return ch
}

func (h *PromoteHandler) fetchUserInfo(uid int64, username string) <-chan *candidate {
	ch := make(chan *candidate)
	go func() {
		defer close(ch)

		if user, err := h.userService.Get(uid); err == nil {
			ch <- &candidate{
				uid:      uid,
				name:     user.Name,
				exist:    true,
				currRole: user.Role,
			}
		} else {
			ch <- &candidate{
				uid:   uid,
				name:  username,
				exist: false,
			}
		}
	}()
	return ch
}

// candidate for promotion to another role
type candidate struct {
	uid      int64
	name     string
	exist    bool
	currRole dto.UserRole
}

func (c candidate) String() string {
	return fmt.Sprintf("candidate(uid=%d, name=%s, exist=%t, currRole=%s)",
		c.uid, c.name, c.exist, c.currRole)
}
