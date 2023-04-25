package dto

// Suggestion is a row in the Suggestions table. It's a post (message) suggested to publication by a user.
type Suggestion struct {
	UID         int64
	MessageID   int
	Anonymously bool
	Published   bool
	Revoked     bool
}
