package repo

import (
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/goSadTgBot/base"
)

// SuggestionService is a repository for the Suggestions table.
type SuggestionService struct {
	appEnv *base.ApplicationEnv
}

func NewSuggestionService(appEnv *base.ApplicationEnv) *SuggestionService {
	return &SuggestionService{appEnv: appEnv}
}

// Get full information about the referenced message.
func (service *SuggestionService) Get(msg *dto.Message) (*dto.Suggestion, error) {
	suggestion := &dto.Suggestion{
		UID:       msg.ChatID,
		MessageID: msg.MessageID,
	}
	err := service.appEnv.Database.QueryRow(service.appEnv.Ctx,
		"SELECT anonymously, published, revoked FROM Suggestions WHERE uid = $1 AND message_id = $2",
		msg.ChatID, msg.MessageID).
		Scan(&suggestion.Anonymously, &suggestion.Published, &suggestion.Revoked)
	return suggestion, err
}

// Create a new suggestion with a reference to the message.
func (service *SuggestionService) Create(msg *dto.Message, anonymously bool) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"INSERT INTO Suggestions(uid, message_id, anonymously) VALUES ($1, $2, $3)",
		msg.ChatID, msg.MessageID, anonymously)
	return err
}

// Publish updates the row. It returns an error if the suggestion was already revoked.
func (service *SuggestionService) Publish(msg *dto.Message) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"UPDATE Suggestions SET published = true WHERE uid = $1 AND message_id = $2",
		msg.ChatID, msg.MessageID)
	return err
}

// Revoke updates the row. It returns an error if the suggestion was already published.
func (service *SuggestionService) Revoke(msg *dto.Message) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"UPDATE Suggestions SET revoked = true WHERE uid = $1 AND message_id = $2",
		msg.ChatID, msg.MessageID)
	return err
}
