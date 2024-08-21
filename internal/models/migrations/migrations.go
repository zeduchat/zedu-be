package migrations

import "github.com/hngprojects/telex_be/internal/models"

// _ = db.AutoMigrate(MigrationModels()...)
func AuthMigrationModels() []interface{} {
	return []interface{}{
		models.NewsLetter{},
		models.TelexSlackChannelMapping{},
		models.SlackTelex{},
		models.ContactUs{},
		models.BlogFeedback{},
		models.BlogCategory{},
		models.Blog{},
		models.User{},
		models.AccessToken{},
		models.Channels{},
		models.Profile{},
		models.UserChannels{},
		models.Message{},
		models.MagicLink{},
		models.PasswordReset{},
		models.Organisation{},
		models.Permission{},
		models.OrgRole{},
		models.LoginActivity{},
		models.Threads{},
		models.ChannelInvitation{},
		models.Invitation{},
		models.Webhook{},
		models.HistoryWebhook{},
		models.SubscriptionPlan{},
		models.HistoryWebhook{},
		models.OrgUserManagement{},
		models.Mentions{},
		models.HelpCenterCategory{},
		models.HelpCenterArticle{},
		models.Integrations{},
		models.IntegrationsSettings{},
	} // an array of db models, example: User{}
}

func AlterColumnModels() []AlterColumn {
	return []AlterColumn{}
}
