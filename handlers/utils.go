package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/logconst"
	log "github.com/sirupsen/logrus"
)

type nameUpdater func(user *tgbotapi.User)

func buildNameUpdater(handlerName string, userService *repo.UserService) nameUpdater {
	return func(user *tgbotapi.User) {
		newName := resolveName(user)
		if err := userService.UpdateName(user.ID, newName); err != nil {
			log.WithField(logconst.FieldHandler, handlerName).
				WithField(logconst.FieldMethod, "Handle").
				WithField(logconst.FieldCalledObject, "UserService").
				WithField(logconst.FieldCalledMethod, "UpdateName").
				WithField("uid", user.ID).
				WithField("name", newName).
				Error("unable to update the name: ", err)
		}
	}
}
