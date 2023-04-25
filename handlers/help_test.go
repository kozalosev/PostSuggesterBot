package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResolveName(t *testing.T) {
	firstName := "foo"
	lastName := "bar"
	userName := "baz"

	fullUser := &tgbotapi.User{
		FirstName: firstName,
		LastName:  lastName,
		UserName:  userName,
	}
	userWithBothNames := &tgbotapi.User{
		FirstName: firstName,
		LastName:  lastName,
	}
	userWithFirstNameOnly := &tgbotapi.User{
		FirstName: firstName,
	}
	userWithoutLastName := &tgbotapi.User{
		FirstName: firstName,
		UserName:  userName,
	}

	assert.Equal(t, userName, resolveName(fullUser))
	assert.Equal(t, fmt.Sprintf("%s %s", firstName, lastName), resolveName(userWithBothNames))
	assert.Equal(t, firstName, resolveName(userWithFirstNameOnly))
	assert.Equal(t, userName, resolveName(userWithoutLastName))
}
