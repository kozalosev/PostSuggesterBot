package repo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func clearDatabase(t *testing.T) {
	for _, table := range []string{"approvals", "suggestions", "users"} {
		_, err := db.Exec(ctx, "DELETE FROM "+table)
		assert.NoError(t, err)
	}
}
