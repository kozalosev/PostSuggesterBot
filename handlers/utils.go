package handlers

import (
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/kozalosev/PostSuggesterBot/db/repo"
	"github.com/kozalosev/goSadTgBot/logconst"
	log "log/slog"
)

type nameUpdater func(user *tgbotapi.User)

func buildNameUpdater(handlerName string, userService *repo.UserService) nameUpdater {
	return func(user *tgbotapi.User) {
		newName := resolveName(user)
		if err := userService.UpdateName(user.ID, newName); err != nil {
			log.Error("unable to update the name",
				logconst.FieldHandler, handlerName,
				logconst.FieldMethod, "Handle",
				logconst.FieldCalledObject, "UserService",
				logconst.FieldCalledMethod, "UpdateName",
				"uid", user.ID,
				"name", newName,
				logconst.FieldError, err)
		}
	}
}
