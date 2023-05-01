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
	if uid == 0 && !autoAdmins {
		return nil
	}

	users := make(tUserKey)
	if uid > 0 {
		users[tUID(uid)] = username
	} else {
		if admins, err := h.fetchAdmins(); err == nil {
			users = admins
		} else {
			log.WithField(logconst.FieldHandler, "PromoteHandler").
				WithField(logconst.FieldMethod, "resolveCandidates").
				WithField(logconst.FieldCalledMethod, "fetchAdmins").
				Error("unable to fetch UIDs of the chat administrators: ", err)
			return nil
		}
	}

	if info, err := h.fetchUsersInfo(users); err == nil {
		return info
	} else {
		log.WithField(logconst.FieldHandler, "PromoteHandler").
			WithField(logconst.FieldMethod, "resolveCandidates").
			WithField(logconst.FieldCalledMethod, "fetchUsersInfo").
			Error("unable to fetch candidates info: ", err)
		return nil
	}
}

// more descriptive type names
type tUID = int64
type tUsername = string
type tUserKey = map[tUID]tUsername

func (h *PromoteHandler) fetchAdmins() (tUserKey, error) {
	channels := funk.Map([]int64{adminChatID, channelID}, func(chatID int64) <-chan *tgbotapi.User {
		return h.fetchAdminsForChat(chatID)
	}).([]<-chan *tgbotapi.User)

	done := make(chan struct{})
	ch, err := harmony.FanIn(done, channels[0], channels[1])
	if err != nil {
		return nil, err
	}

	users := make(map[tUID]tUsername)
	for u := range ch {
		users[u.ID] = resolveName(u)
	}
	return users, nil
}

func (h *PromoteHandler) fetchAdminsForChat(chatID int64) <-chan *tgbotapi.User {
	ch := make(chan *tgbotapi.User)
	go func() {
		defer close(ch)

		reqConfig := tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}}
		if members, err := h.appEnv.Bot.GetStandardAPI().GetChatAdministrators(reqConfig); err == nil {
			for _, m := range members {
				if !m.User.IsBot {
					ch <- m.User
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

func (h *PromoteHandler) fetchUsersInfo(uidToName tUserKey) ([]*candidate, error) {
	uids := keys(uidToName)
	if users, err := h.userService.GetThemAll(uids); err == nil {
		uidsExisting := make(map[int64]struct{})
		candidates := funk.Map(users, func(u *dto.User) *candidate {
			uidsExisting[u.UID] = struct{}{}
			return &candidate{
				uid:      u.UID,
				name:     u.Name,
				exist:    true,
				currRole: u.Role,
			}
		}).([]*candidate)

		uidsNotExisting := funk.FilterInt64(uids, func(uid int64) bool {
			_, ok := uidsExisting[uid]
			return !ok
		})
		for _, uid := range uidsNotExisting {
			c := &candidate{
				uid:   uid,
				name:  uidToName[uid],
				exist: false,
			}
			candidates = append(candidates, c)
		}
		return candidates, nil
	} else {
		return nil, err
	}
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
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
