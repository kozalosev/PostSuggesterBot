package handlers

import (
	"github.com/kozalosev/goSadTgBot/logconst"
	log "log/slog"
	"os"
	"strconv"
)

const (
	success   = "success"
	failure   = "failure"
	duplicate = "duplicate"

	callbackDataSep = ":"
)

func parseNotUserID(idEnv string) int64 {
	idStr := os.Getenv(idEnv)
	if len(idStr) >= 3 && idStr[:3] != "-100" {
		idStr = "-100" + idStr
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logconst.FailedToParseEnvironmentVariableWithContext(
			log.With(logconst.FieldFunc, "parseNotUserID"),
			idEnv, err)
	}
	return id
}
