package repo

import (
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/goSadTgBot/settings"
	"github.com/stretchr/testify/assert"
	"testing"
)

var userService = NewUserService(appEnv)

func TestUserService_CreateAndGet(t *testing.T) {
	clearDatabase(t)
	createTestUser(t)

	user := getTestUser(t)
	assert.Equal(t, TestUID, user.UID)
	assert.Equal(t, TestUser, user.Name)
}

func TestUserService_BanAndUnban(t *testing.T) {
	clearDatabase(t)
	createTestUser(t)

	user := getTestUser(t)
	assert.False(t, user.Banned)

	assert.NoError(t, userService.Ban(user.UID))

	user = getTestUser(t)
	assert.True(t, user.Banned)

	assert.NoError(t, userService.Unban(user.UID))

	user = getTestUser(t)
	assert.False(t, user.Banned)
}

func TestUserService_UpdateName(t *testing.T) {
	clearDatabase(t)
	createTestUser(t)

	user := getTestUser(t)
	assert.Equal(t, TestUser, user.Name)

	newName := "foobar"
	assert.NoError(t, userService.UpdateName(user.UID, newName))

	user = getTestUser(t)
	assert.Equal(t, newName, user.Name)
}

func TestUserService_ChangeLanguage(t *testing.T) {
	clearDatabase(t)
	createTestUser(t)

	defaultLang := "defLang"
	lang, _ := userService.FetchUserOptions(TestUID, defaultLang)
	assert.Equal(t, settings.LangCode(defaultLang), lang)

	var newLang settings.LangCode = "ru"
	assert.NoError(t, userService.ChangeLanguage(TestUID, newLang))

	lang, _ = userService.FetchUserOptions(TestUID, defaultLang)
	assert.Equal(t, newLang, lang)
}

func TestUserService_Promote(t *testing.T) {
	_, opts := userService.FetchUserOptions(TestUID, "")
	assert.IsType(t, &dto.UserOptions{}, opts)
	assert.Equal(t, dto.UsualUser, opts.(*dto.UserOptions).Role)

	assert.NoError(t, userService.Promote(TestUID, dto.Author))

	_, opts = userService.FetchUserOptions(TestUID, "")
	assert.Equal(t, dto.Author, opts.(*dto.UserOptions).Role)
}

func createTestUser(t *testing.T) {
	assert.NoError(t, userService.Create(TestUID, TestUser))
}

func getTestUser(t *testing.T) *dto.User {
	user, err := userService.Get(TestUID)
	assert.NoError(t, err)
	return user
}
