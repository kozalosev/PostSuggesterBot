package repo

import (
	"github.com/kozalosev/PostSuggesterBot/db/dto"
	"github.com/kozalosev/goSadTgBot/base"
	"github.com/kozalosev/goSadTgBot/logconst"
	log "github.com/sirupsen/logrus"
)

// ApprovalService is a repository for the Approvals table.
type ApprovalService struct {
	appEnv *base.ApplicationEnv
}

func NewApprovalService(appEnv *base.ApplicationEnv) *ApprovalService {
	return &ApprovalService{appEnv: appEnv}
}

// GetApprovers retrieves the list of all authors approved the message.
func (service *ApprovalService) GetApprovers(msg *dto.Message) ([]string, error) {
	rows, err := service.appEnv.Database.Query(service.appEnv.Ctx,
		"SELECT u.name FROM Approvals a JOIN Users u ON a.approved_by = u.uid WHERE author_uid = $1 AND message_id = $2",
		msg.ChatID, msg.MessageID)
	if err != nil {
		return nil, err
	}

	var (
		approver  string
		approvers []string
	)
	for rows.Next() {
		if err = rows.Scan(&approver); err == nil {
			approvers = append(approvers, approver)
		} else {
			log.WithField(logconst.FieldService, "ApprovalService").
				WithField(logconst.FieldMethod, "GetApprovers").
				WithField(logconst.FieldCalledObject, "Rows").
				WithField(logconst.FieldCalledMethod, "Scan").
				Error(err)
		}
	}
	return approvers, err
}

// Approve creates a new approval in the database.
func (service *ApprovalService) Approve(msg *dto.Message, approverUID int64) error {
	_, err := service.appEnv.Database.Exec(service.appEnv.Ctx,
		"INSERT INTO Approvals(author_uid, message_id, approved_by) VALUES ($1, $2, $3)",
		msg.ChatID, msg.MessageID, approverUID)
	return err
}
