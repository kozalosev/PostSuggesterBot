package repo

import (
	"errors"
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	"github.com/kozalosev/goSadTgBot/settings"
	log "github.com/sirupsen/logrus"
)

var NoRowsWereAffected = errors.New("no rows were affected")

// UserService is a repository for the Users table.
type UserService struct {
	appEnv *base.ApplicationEnv
}

func NewUserService(appEnv *base.ApplicationEnv) *UserService {
	return &UserService{appEnv: appEnv}
}

// FetchUserOptions is the implementation of the [settings.OptionsFetcher.FetchUserOptions] method for this application.
func (service *UserService) FetchUserOptions(uid int64, defaultLang string) (settings.LangCode, settings.UserOptions) {
	var (
		language *string
		opts     dto.UserOptions
	)
	if err := service.appEnv.Database.QueryRow(service.appEnv.Ctx,
		"SELECT language, banned, role FROM Users WHERE uid = $1", uid).
		Scan(&language, &opts.Banned, &opts.Role); err != nil {

		log.WithField(logconst.FieldService, "UserService").
			WithField(logconst.FieldMethod, "FetchUserOptions").
			WithField(logconst.FieldCalledObject, "Row").
			WithField(logconst.FieldCalledMethod, "Scan").
			Error(err)
	}
	if language == nil {
		language = &defaultLang
	}
	return settings.LangCode(*language), &opts
}

func (service *UserService) Get(uid int64) (*dto.User, error) {
	var user dto.User
	err := service.appEnv.Database.QueryRow(service.appEnv.Ctx,
		"SELECT uid, name, banned, role FROM Users WHERE uid = $1", uid).
		Scan(&user.UID, &user.Name, &user.Banned, &user.Role)
	return &user, err
}

// Create a new user in the database. The NoRowsWereAffected error will be returned if he already exists.
func (service *UserService) Create(uid int64, name string) error {
	tag, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"INSERT INTO Users(uid, name) VALUES ($1, $2) ON CONFLICT DO NOTHING", uid, name)
	if err == nil && tag.RowsAffected() < 1 {
		return NoRowsWereAffected
	} else {
		return err
	}
}

func (service *UserService) UpdateName(uid int64, name string) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"UPDATE Users SET name = $2 WHERE uid = $1", uid, name)
	return err
}

func (service *UserService) ChangeLanguage(uid int64, lang settings.LangCode) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"UPDATE Users SET language = $2 WHERE uid = $1", uid, lang)
	return err
}

func (service *UserService) Ban(uid int64) error {
	return service.changeBanValue(uid, true)
}

func (service *UserService) Unban(uid int64) error {
	return service.changeBanValue(uid, false)
}

func (service *UserService) changeBanValue(uid int64, ban bool) error {
	tag, err := service.appEnv.Database.Exec(service.appEnv.Ctx, "UPDATE Users SET banned = $2 WHERE uid = $1", uid, ban)
	if err == nil && tag.RowsAffected() < 1 {
		return NoRowsWereAffected
	} else {
		return err
	}
}

// Promote is the method to change the role of some user.
func (service *UserService) Promote(uid int64, role dto.UserRole) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx, "UPDATE Users SET role = $2 WHERE uid = $1", uid, role)
	return err
}
