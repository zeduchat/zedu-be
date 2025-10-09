package models

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

type AgentJobArgs struct {
	AgentIds []string `json:"agent_ids"`
	OrgId    string   `json:"org_id"`
	UserId   string   `json:"user_id"`
	Messages map[string]string
}

func (AgentJobArgs) Kind() string { return "agent_job" }

func (w *AgentJobArgs) InsertAgentJob(ctx context.Context) error {
	client := storage.DB.River
	_, err := client.Insert(ctx, w, &river.InsertOpts{
		MaxAttempts: 5,
		ScheduledAt: time.Now().Add(time.Minute),
		Priority:    1,
	})

	return err
}
