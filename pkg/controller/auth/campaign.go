package auth

import (
	"github.com/gin-gonic/gin"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"github.com/hngprojects/telex_be/services/campaign"
)

func (base *Controller) recordCampaign(a *contract.Attribution, result gin.H) {
	if result["is_new_user"] != true {
		return
	}
	user, ok := result["user"].(map[string]any)
	if !ok {
		return
	}
	id, ok := user["id"].(string)
	if !ok || id == "" {
		return
	}
	if err := campaign.Record(base.Db.Postgresql, a, "sign_up", id); err != nil {
		base.Logger.Error("campaign measurement unavailable", err)
	}
}
