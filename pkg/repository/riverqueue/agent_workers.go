package riverqueueBg

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

type AgentJobWorker struct {
	Logger *utility.Logger
	river.WorkerDefaults[models.AgentJobArgs]
}

func (w *AgentJobWorker) Work(ctx context.Context, job *river.Job[models.AgentJobArgs]) error {

	var (
		dmChan models.DmChannels
	)

	agentJobs := job.Args
	logger := w.Logger

	for _, agentId := range agentJobs.AgentIds {

		dmChan.ParticipantId = &agentId
		dmChan.UserId = agentJobs.UserId
		dmChan.OrgId = agentJobs.OrgId
		dmChan.ChatType = "bot"
		dmChan.ChannelId = utility.GenerateUUID()
		dmChan.ID = utility.GenerateUUID()

		resp, err := dmChan.CreateAgentDMChannel(storage.DB.Postgresql)

		if err != nil {
			logger.Error("An error occured while creating channel with user, error: %v,", err)
			return err
		}

		logger.Info("Created agent channel successfully")

		channel_id := resp.ID

		req := models.BotReturnRequest{
			ThreadId:  utility.GenerateUUID(),
			Content:   agentJobs.Messages[agentId],
			UserId:    agentJobs.UserId,
			OrgId:     agentJobs.OrgId,
			AgentId:   agentId,
			ChannelID: channel_id,
		}

		_, code, err := dm.BotResponse(req, storage.DB, logger)

		if err != nil {
			logger.Error("An error occured while sending message to user, error: %v, with code :%v", err, code)
			return err
		}

		logger.Info("Successfully sent message to user from agent task, agentId: %vv, userId: %v.........", agentId, agentJobs.UserId)

	}

	logger.Info("Successfully sent all messages to user from agent task.........")

	return nil
}
